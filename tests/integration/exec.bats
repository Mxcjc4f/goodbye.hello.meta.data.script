#!/usr/bin/env bats

load helpers

function setup() {
	setup_busybox
}

function teardown() {
	teardown_bundle
}

@test "runc exec" {
	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec test_busybox echo Hello from exec
	[ "$status" -eq 0 ]
	echo text echoed = "'""${output}""'"
	[[ "${output}" == *"Hello from exec"* ]]
}

@test "runc exec --pid-file" {
	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec --pid-file pid.txt test_busybox echo Hello from exec
	[ "$status" -eq 0 ]
	echo text echoed = "'""${output}""'"
	[[ "${output}" == *"Hello from exec"* ]]

	# check pid.txt was generated
	[ -e pid.txt ]

	output=$(cat pid.txt)
	[[ "$output" =~ [0-9]+ ]]
	[[ "$output" != $(__runc state test_busybox | jq '.pid') ]]
}

@test "runc exec --pid-file with new CWD" {
	bundle="$(pwd)"
	# create pid_file directory as the CWD
	mkdir pid_file
	cd pid_file

	# run busybox detached
	runc run -d -b "$bundle" --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec --pid-file pid.txt test_busybox echo Hello from exec
	[ "$status" -eq 0 ]
	echo text echoed = "'""${output}""'"
	[[ "${output}" == *"Hello from exec"* ]]

	# check pid.txt was generated
	[ -e pid.txt ]

	output=$(cat pid.txt)
	[[ "$output" =~ [0-9]+ ]]
	[[ "$output" != $(__runc state test_busybox | jq '.pid') ]]
}

@test "runc exec ls -la" {
	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec test_busybox ls -la
	[ "$status" -eq 0 ]
	[[ ${lines[0]} == *"total"* ]]
	[[ ${lines[1]} == *"."* ]]
	[[ ${lines[2]} == *".."* ]]
}

@test "runc exec ls -la with --cwd" {
	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec --cwd /bin test_busybox pwd
	[ "$status" -eq 0 ]
	[[ ${output} == "/bin"* ]]
}

@test "runc exec --env" {
	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec --env RUNC_EXEC_TEST=true test_busybox env
	[ "$status" -eq 0 ]

	[[ ${output} == *"RUNC_EXEC_TEST=true"* ]]
}

@test "runc exec --user" {
	# --user can't work in rootless containers that don't have idmap.
	[[ "$ROOTLESS" -ne 0 ]] && requires rootless_idmap

	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	runc exec --user 1000:1000 test_busybox id
	[ "$status" -eq 0 ]

	[[ "${output}" == "uid=1000 gid=1000"* ]]
}

@test "runc exec --additional-gids" {
	requires root

	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	wait_for_container 15 1 test_busybox

	runc exec --user 1000:1000 --additional-gids 100 --additional-gids 65534 test_busybox id -G
	[ "$status" -eq 0 ]

	[[ ${output} == "1000 100 65534" ]]
}

@test "runc exec --preserve-fds" {
	# run busybox detached
	runc run -d --console-socket "$CONSOLE_SOCKET" test_busybox
	[ "$status" -eq 0 ]

	echo hello >preserve-fds.test
	# fd 3 is used by bats, so we use 4
	exec 4<preserve-fds.test
	runc exec --preserve-fds=2 test_busybox cat /proc/self/fd/4
	[ "$status" -eq 0 ]
	[[ "${output}" == "hello" ]]
}

function check_exec_debug() {
	[[ "$*" == *"nsexec container setup"* ]]
	[[ "$*" == *"child process in init()"* ]]
	[[ "$*" == *"setns_init: about to exec"* ]]
}

@test "runc --debug exec" {
	runc run -d --console-socket "$CONSOLE_SOCKET" test
	[ "$status" -eq 0 ]

	runc --debug exec test true
	[ "$status" -eq 0 ]
	[[ "${output}" == *"level=debug"* ]]
	check_exec_debug "$output"
}

@test "runc --debug --log exec" {
	runc run -d --console-socket "$CONSOLE_SOCKET" test
	[ "$status" -eq 0 ]

	runc --debug --log log.out exec test true
	# check output does not include debug info
	[[ "${output}" != *"level=debug"* ]]

	cat log.out >&2
	# check expected debug output was sent to log.out
	output=$(cat log.out)
	[[ "${output}" == *"level=debug"* ]]
	check_exec_debug "$output"
}
