package cfgutils

import "errors"

var (
	userdataSampleContent = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
`

	userdataSampleContent1rc = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
`

	userdataSampleContentSsh = `#cloud-config
ssh_authorized_keys:
  - ssh-rsa AAASampleContent
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

	inputOneItemSshAuthKeys = []string{
		`ssh-rsa AAAAB3NzaC1yc`,
	}
	inputTwoItemsSshAuthKeys = []string{
		`ssh-rsa AAAAitem1`,
		`ssh-rsa AAAAitem2`,
	}

	expectedSuseProducts1rc1wf = `#cloud-config
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/register-suse-modules.sh
    content: H4sIAAAAAAAA/wAAAP//bFRhb9s2EP1s/opXVZ6TbZLnZJ9aOFhRdFsHLBnmDsPQFAEtnmJmEqkcT26NxP99oCRrcTZ9EUG+e3fv8UkvX8zX1s3XOmxUIEFGSmxNQXTdLNMTo4XwzfSvbFpnU3NzM/35ZvrrzXR1qugLFbhAMpe6mTPd2iDEWWgDZbU3bUUhSx9Gqn1e+dsEZxdfLZSyJV6g8HWtnUG2xd19d4CLuaHt3LVV9RqyIacAKjYeyWoXhGpcrdA3Yi3WOxTaOS9o2BdEJsfs7n524M0q6wi/rK4u+/MQPMMGMN23lslg3QpidelbZ+Addr5lhK7TQFW3QbAmWBdEVxUZ6IBGs8CXccA4kK31LaFhavQwVRBqQo7fKtKBwFQSQ3yHbwMxau1aXaH0DEOibeSNHbgtYn2Is/R81t32U4191lR6poPieP7ZyubIlRxv1p4lnh1tJ9HNL1awUKVVajBWdA9d/bF6h9+f4vM8UaqozTKJZ2+9c1QIMsZiscjOzs6y8/NzZIQ7v3E/GE/JQJle4FnB1/E5hqZFbZQKrfHH4CyIljbgMWYiY8zyj5/wiOT6JLeGnNjSEp9ey/VJ3iP7teZi06+2xMF6d5rM8IjPG1sR3v+4Wqaza5mBSZvIag2GPrEQW+LXMH5MmzXL1Jp5uiWepxHx7QBfpv07WmlLfERy2MByieTSy2AhMZkEn8YUH5jfvP3w/ury1YiKzqfWJGoS02BhHR4Wef79fpinf4ZaEaobQWq76PBzihH+Eh94FxN3gHTR679JNZn890obpA/W7Ofpw5Y4vqLmfaImk/FG430lYwdbdhtP5B2PumqL+MGVbVXtoAuxWy1k+jGfotdM+u9/ZVaB/ofsJy94x+y5k51ak+NPbQXx38EkvDvm7OSFiqjB+XdRw3MRk8mkS9+hoLTd0nhH0cNxisH1KoZmN3pJ5tVBR2lVLPonAAD//6iMnDdBBQAA
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedSuseProduct2rc1wf = `#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/register-suse-modules.sh
    content: H4sIAAAAAAAA/wAAAP//bFRhb9s2EP1s/opXVZ6TbZLnZJ9aOFhRdFsHLBnmDsPQFAEtnmJmEqkcT26NxP99oCRrcTZ9EUG+e3fv8UkvX8zX1s3XOmxUIEFGSmxNQXTdLNMTo4XwzfSvbFpnU3NzM/35ZvrrzXR1qugLFbhAMpe6mTPd2iDEWWgDZbU3bUUhSx9Gqn1e+dsEZxdfLZSyJV6g8HWtnUG2xd19d4CLuaHt3LVV9RqyIacAKjYeyWoXhGpcrdA3Yi3WOxTaOS9o2BdEJsfs7n524M0q6wi/rK4u+/MQPMMGMN23lslg3QpidelbZ+Addr5lhK7TQFW3QbAmWBdEVxUZ6IBGs8CXccA4kK31LaFhavQwVRBqQo7fKtKBwFQSQ3yHbwMxau1aXaH0DEOibeSNHbgtYn2Is/R81t32U4191lR6poPieP7ZyubIlRxv1p4lnh1tJ9HNL1awUKVVajBWdA9d/bF6h9+f4vM8UaqozTKJZ2+9c1QIMsZiscjOzs6y8/NzZIQ7v3E/GE/JQJle4FnB1/E5hqZFbZQKrfHH4CyIljbgMWYiY8zyj5/wiOT6JLeGnNjSEp9ey/VJ3iP7teZi06+2xMF6d5rM8IjPG1sR3v+4Wqaza5mBSZvIag2GPrEQW+LXMH5MmzXL1Jp5uiWepxHx7QBfpv07WmlLfERy2MByieTSy2AhMZkEn8YUH5jfvP3w/ury1YiKzqfWJGoS02BhHR4Wef79fpinf4ZaEaobQWq76PBzihH+Eh94FxN3gHTR679JNZn890obpA/W7Ofpw5Y4vqLmfaImk/FG430lYwdbdhtP5B2PumqL+MGVbVXtoAuxWy1k+jGfotdM+u9/ZVaB/ofsJy94x+y5k51ak+NPbQXx38EkvDvm7OSFiqjB+XdRw3MRk8mkS9+hoLTd0nhH0cNxisH1KoZmN3pJ5tVBR2lVLPonAAD//6iMnDdBBQAA
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedSuseProduct1rc2wf = `#cloud-config
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /tmp/register-suse-modules.sh
    content: H4sIAAAAAAAA/wAAAP//bFRhb9s2EP1s/opXVZ6TbZLnZJ9aOFhRdFsHLBnmDsPQFAEtnmJmEqkcT26NxP99oCRrcTZ9EUG+e3fv8UkvX8zX1s3XOmxUIEFGSmxNQXTdLNMTo4XwzfSvbFpnU3NzM/35ZvrrzXR1qugLFbhAMpe6mTPd2iDEWWgDZbU3bUUhSx9Gqn1e+dsEZxdfLZSyJV6g8HWtnUG2xd19d4CLuaHt3LVV9RqyIacAKjYeyWoXhGpcrdA3Yi3WOxTaOS9o2BdEJsfs7n524M0q6wi/rK4u+/MQPMMGMN23lslg3QpidelbZ+Addr5lhK7TQFW3QbAmWBdEVxUZ6IBGs8CXccA4kK31LaFhavQwVRBqQo7fKtKBwFQSQ3yHbwMxau1aXaH0DEOibeSNHbgtYn2Is/R81t32U4191lR6poPieP7ZyubIlRxv1p4lnh1tJ9HNL1awUKVVajBWdA9d/bF6h9+f4vM8UaqozTKJZ2+9c1QIMsZiscjOzs6y8/NzZIQ7v3E/GE/JQJle4FnB1/E5hqZFbZQKrfHH4CyIljbgMWYiY8zyj5/wiOT6JLeGnNjSEp9ey/VJ3iP7teZi06+2xMF6d5rM8IjPG1sR3v+4Wqaza5mBSZvIag2GPrEQW+LXMH5MmzXL1Jp5uiWepxHx7QBfpv07WmlLfERy2MByieTSy2AhMZkEn8YUH5jfvP3w/ury1YiKzqfWJGoS02BhHR4Wef79fpinf4ZaEaobQWq76PBzihH+Eh94FxN3gHTR679JNZn890obpA/W7Ofpw5Y4vqLmfaImk/FG430lYwdbdhtP5B2PumqL+MGVbVXtoAuxWy1k+jGfotdM+u9/ZVaB/ofsJy94x+y5k51ak+NPbQXx38EkvDvm7OSFiqjB+XdRw3MRk8mkS9+hoLTd0nhH0cNxisH1KoZmN3pJ5tVBR2lVLPonAAD//6iMnDdBBQAA
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedStr2Cmd = `
#cloud-config
runcmd:
  - timedatectl set-timezone Europe/Warsaw
  - echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log
`

	expectedStr1Ssh = `
#cloud-config
ssh_authorized_keys:
  - ssh-rsa AAAAB3NzaC1yc
`

	expectedStr2Ssh = `
#cloud-config
ssh_authorized_keys:
  - ssh-rsa AAASampleContent
  - ssh-rsa AAAAB3NzaC1yc
`

	expectedStr3Ssh = `
#cloud-config
ssh_authorized_keys:
  - ssh-rsa AAASampleContent
  - ssh-rsa AAAAitem1
  - ssh-rsa AAAAitem2
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

	inputOneItemWriteFiles = []CloudConfigItemWriteFiles{
		NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files")}

	inputTwoItemsWriteFiles = []CloudConfigItemWriteFiles{
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

	input1ItemRunCmdCast = cloudInitFile{
		RunCmds: []string{`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`}}

	input1ItemRunCmdCast1ItemWriteFiles = cloudInitFile{
		RunCmds: []string{`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`},
		WriteFiles: []CloudConfigItemWriteFiles{
			NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
		},
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

	input2ItemsRunCmdCast2ItemsWriteFiles = cloudInitFile{
		RunCmds: []string{`echo "Boot completed at $(date)" >> /tmp/cloud-config-test-runcmd.log`,
			`echo "Cloud config test succeeded" >> /tmp/cloud-config-test-runcmd.log`},
		WriteFiles: []CloudConfigItemWriteFiles{
			NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files.log", "Cloud config succeeded for write_files"),
			NewCloudConfigItemWriteFiles("/tmp/cloud-config-test-write-files-2.log", "Cloud config succeeded for write_files part 2"),
		},
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

	expectedStrDisablePwdAuth = `
#cloud-config
ssh_pwauth: false
write_files:
  - path: /etc/ssh/sshd_config.d/30-auth-methods.conf
    content: H4sIAAAAAAAA/wAAAP//ciwtyUjNK8lMTizJzM/zTS3JyE8pVigoTcrJTM5OrQQEAAD//5RFpK0fAAAA
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedStrDisablePwdAuth1rc = `#cloud-config
ssh_pwauth: false
runcmd:
  - timedatectl set-timezone Europe/Warsaw
write_files:
  - path: /etc/ssh/sshd_config.d/30-auth-methods.conf
    content: H4sIAAAAAAAA/wAAAP//ciwtyUjNK8lMTizJzM/zTS3JyE8pVigoTcrJTM5OrQQEAAD//5RFpK0fAAAA
    encoding: "gzip+b64"
    permissions: "0644"
`

	sampleDisablePwdAuthExists = `
#cloud-config
ssh_pwauth: true
`

	sample1rc2wf = `#cloud-config
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`

	sampleHostname1rc2wf = `#cloud-config
hostname: alpha1
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedDisablePwdAuth1rc2wf = `#cloud-config
ssh_pwauth: false
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
  - path: /etc/ssh/sshd_config.d/30-auth-methods.conf
    content: H4sIAAAAAAAA/wAAAP//ciwtyUjNK8lMTizJzM/zTS3JyE8pVigoTcrJTM5OrQQEAAD//5RFpK0fAAAA
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedDisablePwdAuth1rc1wf = `#cloud-config
ssh_pwauth: false
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedStr2rc1wf = `#cloud-config
hostname: alpha1
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedBoolStr2rc1wf = `#cloud-config
hostname: alpha1
ssh_pwauth: false
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedBoolStrInt2rc1wf = `#cloud-config
hostname: alpha1
ssh_pwauth: false
boot_cmd_timeout: 30
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`

	expectedInt2rc1wf = `#cloud-config
boot_cmd_timeout: 30
runcmd:
  - sh /tmp/register-suse-modules.sh
  - rm /tmp/register-suse-modules.sh
write_files:
  - path: /tmp/foo
    content: Foo was here
    encoding: "gzip+b64"
    permissions: "0644"
`
)

const sampleRke2ConfigFileContent = `kubelet-arg+: "provider-id=fsas-cdi://%s"`
