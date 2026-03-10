package cfgutils

import "errors"

var (
	userdataSampleContent1rc = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
`

	userdataSampleContent1wf = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	userdataSampleContentNoSections                 = `#cloud-config`
	userdataSampleInvalidYamlContentRandomAscii     = `.32??#(&&)58ffo:bar`
	userdataSampleInvalidYamlContentRunCmdIsInteger = `#cloud-config
  runcmd: 123`
	userdataSampleInvalidYamlContentRunCmdIsString = `#cloud-config
  runcmd: foobar`
	userdataSampleInvalidYamlContentRunCmdIsBool = `#cloud-config
  runcmd: true`
	userdataSampleInvalidYamlContentRunCmdIsMap = `#cloud-config
  runcmd:
    foo: bar`
	userdataSampleInvalidYamlContentRunCmdIsNil = `#cloud-config
  runcmd:`

	inputOneItemRunCmd = []string{
		`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
	}

	expectedStr2Cmd = `
#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
`

	inputTwoItemsRunCmd = []string{
		`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
		`echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log`,
	}

	expectedStr3Cmd = `
#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
  - echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log
`
	expectedErrorReadingFromFile = errors.New("error while reading file")

	expectedStr1Cmd = `
#cloud-config
runcmd:
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
`

	expectedErrorWritingToFile = errors.New("error while writing file")

	userdataSampleContentWriteFiles = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	inputOneItemWriteFiles = []CloudConfigItem{
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files")}

	inputTwoItemsWriteFiles = []CloudConfigItem{
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files-2.log", "Cloud config succeeded for write_files part 2"),
	}

	expectedStr1Write = `#cloud-config
write_files:
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedStr2Write = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedStr2WriteExe = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/run.sh
    content: H4sIAAAAAAAA/wAAAP//UlbUT8rM009KLM4ABAAA//9pWODrCwAAAA==
    encoding: "gzip+b64"
    permissions: "0744"`

	expectedStr2WriteSetPermissions = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cdi.cert
    content: H4sIAAAAAAAA/wAAAP//UlZWTkpNz8xTSE4tKgEEAAD//wyPJqgNAAAA
    encoding: "gzip+b64"
    permissions: "0400"`

	expectedStr3Write = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files-2.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS1WKEgsKlEwAgQAAP//55tZZi0AAAA=
    encoding: "gzip+b64"
    permissions: "0644"`

	userdataSampleContentBothSections = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	input1ItemRunCmdCast = []CloudConfigItem{
		NewCloudConfigItemRunCmd([]string{`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`})}

	input1ItemRunCmdCast1ItemWriteFiles = []CloudConfigItem{
		NewCloudConfigItemRunCmd([]string{`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`}),
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
	}

	expectedStr2Cmd1Write = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedStr2Cmd2Write = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"`

	input2ItemsRunCmdCast2ItemsWriteFiles = []CloudConfigItem{
		NewCloudConfigItemRunCmd([]string{
			`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
			`echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log`}),
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files-2.log", "Cloud config succeeded for write_files part 2"),
	}

	expectedStr3Cmd3Write = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
  - echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files-2.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS1WKEgsKlEwAgQAAP//55tZZi0AAAA=
    encoding: "gzip+b64"
    permissions: "0644"`

	userdataSampleContentCmdNoWriteYes = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedStr1Cmd2Write = `#cloud-config
runcmd:
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"`

	userdataSampleContentCmdYesWriteNo = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw`

	expectedStr2Cmd1WriteBis = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedStr1Cmd1Write = `#cloud-config
runcmd:
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
write_files:
  - path: /tmp/cloud-config-test-write-files.log
    content: H4sIAAAAAAAA/wAAAP//cs7JL01RSM7PS8tMVyguTU5OTU1JTVFIyy9SKC/KLEmNT8vMSS0GBAAA//84FqCbJgAAAA==
    encoding: "gzip+b64"
    permissions: "0644"`

	sampleRke2ConfigName         = "100-fsas-providerid.yaml"
	expectedImplantRke2Config2wf = `#cloud-config
write_files:
  - path: /etc/rancher/k3s/config.yaml.d/100-fsas-providerid.yaml
    content: H4sIAAAAAAAA/wAAAP//BMBhCoUgDADgqzz8+xjm0lChw0y3hRQUWp2/b3+KHHID9e2ff+bq59tYOjReddCAyi1b62JCrmGBuZCAl0CQSCdArcSk6B1G8wUAAP//2sQU+UsAAAA=
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /etc/rancher/rke2/config.yaml.d/100-fsas-providerid.yaml
    content: H4sIAAAAAAAA/wAAAP//BMBhCoUgDADgqzz8+xjm0lChw0y3hRQUWp2/b3+KHHID9e2ff+bq59tYOjReddCAyi1b62JCrmGBuZCAl0CQSCdArcSk6B1G8wUAAP//2sQU+UsAAAA=
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedImplantRke2Config1rc2wf = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
write_files:
  - path: /etc/rancher/k3s/config.yaml.d/100-fsas-providerid.yaml
    content: H4sIAAAAAAAA/wAAAP//BMBhCoUgDADgqzz8+xjm0lChw0y3hRQUWp2/b3+KHHID9e2ff+bq59tYOjReddCAyi1b62JCrmGBuZCAl0CQSCdArcSk6B1G8wUAAP//2sQU+UsAAAA=
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /etc/rancher/rke2/config.yaml.d/100-fsas-providerid.yaml
    content: H4sIAAAAAAAA/wAAAP//BMBhCoUgDADgqzz8+xjm0lChw0y3hRQUWp2/b3+KHHID9e2ff+bq59tYOjReddCAyi1b62JCrmGBuZCAl0CQSCdArcSk6B1G8wUAAP//2sQU+UsAAAA=
    encoding: "gzip+b64"
    permissions: "0644"`

	expectedImplantRke2Config3wf = `#cloud-config
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /etc/rancher/k3s/config.yaml.d/100-fsas-providerid.yaml
    content: H4sIAAAAAAAA/wAAAP//BMBhCoUgDADgqzz8+xjm0lChw0y3hRQUWp2/b3+KHHID9e2ff+bq59tYOjReddCAyi1b62JCrmGBuZCAl0CQSCdArcSk6B1G8wUAAP//2sQU+UsAAAA=
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /etc/rancher/rke2/config.yaml.d/100-fsas-providerid.yaml
    content: H4sIAAAAAAAA/wAAAP//BMBhCoUgDADgqzz8+xjm0lChw0y3hRQUWp2/b3+KHHID9e2ff+bq59tYOjReddCAyi1b62JCrmGBuZCAl0CQSCdArcSk6B1G8wUAAP//2sQU+UsAAAA=
    encoding: "gzip+b64"
    permissions: "0644"`
)

const sampleRke2ConfigFileContent = `kubelet-arg+: "provider-id=fsas-cdi://%s"`
