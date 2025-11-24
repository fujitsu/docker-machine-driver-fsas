package cfgutils

import "errors"

var (
	userdataSampleContent = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	userdataSampleContentNoSectionRunCmd = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	inputOneItemRunCmd = []string{
		`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
	}

	expectedStr2Cmd1Write = `
#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	inputTwoItemsRunCmd = []string{
		`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
		`echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log`,
	}

	expectedStr3Cmd1Write = `
#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
  - echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedErrorReadingFromFile = errors.New("error while reading file")

	expectedStr1Cmd1Write = `
#cloud-config
runcmd:
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedErrorWritingToFile = errors.New("error while writing file")
)
