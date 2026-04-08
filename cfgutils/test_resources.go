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
    content: H4sIAAAAAAAA/wAAAP//bFRhb9s2EP1s/opXVZ6TbZLmZJ9aOFhRdFsHLBnmDsPQFAEtnmJmEqkcT26NJP99oCQ7cVZ+EcF79+7e8VEvXxQr64qVDmsVSJCREttQEN20i/TIaCF8N/0nmzbZ1FxdTX+9mv5+NV0eK/pCJc6QFNK0BdO1DUKchS5Q1njT1RSy9G5P9ZDX/jrBydk3c6VshRcofdNoZ5BtcHPbB3BWGNoUrqvr15A1OQVQufZIltsg1OBiiaEQa7HeodTOeUHLviQyOWY3t7Mdb1ZbR/hteXE+xEPwDBvAdNtZJoNVJ4jZle+cgXfY+o4R+kojVdMFwYpgXRBd12SgA1rNAl/FBmNDttHXhJap1WNXQagNOf6oSQcCU0UM8T2+C8RotOt0jcozDIm2kTdW4K6M+SH2MvBZdz10ta+zosoz7RTH+Gcr64Op5Hiz8iwxdnCcxGl+sYK5qqxS42BFD9DlX8t3+PMpPs8TpcrGLJIYe+udo1KQMebzeXZycpKdnp4iI9z4tfvJeEpGyvQMzxK+jesQmpaNUSp0xh+CsyBauoD76ImMMcs/fsI9ksuj3BpyYitLfHwpl0f5gBz2msv1sNsQB+vdcTLDPT6vbU14//Nykc4uZQYmbSKrNRjrxERsiF/D+L3brFmk1hTphrhII+L7Eb5Ih28cpa3wEcnuAIsFknMv4wiJyST4tHfxjvnN2w/vL85f7VFx8qk1iZpEN1hYh7t5nv/4MPYzrDFXhJpWkNreOvycYofeb17iA2+j83bQ3oLD21STyf+vtkV6Z81Dkd5tiOMnan9I1GSyv9l4b4+lbNUfPJF52PKyK+PDq7q63kKXYjdayBy2G9eKSf/7KLcO9BWyX7zgHbPnXn5qTY6/tRXEfwiT8PaQs5cXaqIWpz9EDc9FTCaT3oW7hMr2W+MdKfWki3H6dTTPdj9LMq92OiqrYtJ/AQAA///S8LvxSQUAAA==
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
    content: H4sIAAAAAAAA/wAAAP//bFRhb9s2EP1s/opXVZ6TbZLmZJ9aOFhRdFsHLBnmDsPQFAEtnmJmEqkcT26NJP99oCQ7cVZ+EcF79+7e8VEvXxQr64qVDmsVSJCREttQEN20i/TIaCF8N/0nmzbZ1FxdTX+9mv5+NV0eK/pCJc6QFNK0BdO1DUKchS5Q1njT1RSy9G5P9ZDX/jrBydk3c6VshRcofdNoZ5BtcHPbB3BWGNoUrqvr15A1OQVQufZIltsg1OBiiaEQa7HeodTOeUHLviQyOWY3t7Mdb1ZbR/hteXE+xEPwDBvAdNtZJoNVJ4jZle+cgXfY+o4R+kojVdMFwYpgXRBd12SgA1rNAl/FBmNDttHXhJap1WNXQagNOf6oSQcCU0UM8T2+C8RotOt0jcozDIm2kTdW4K6M+SH2MvBZdz10ta+zosoz7RTH+Gcr64Op5Hiz8iwxdnCcxGl+sYK5qqxS42BFD9DlX8t3+PMpPs8TpcrGLJIYe+udo1KQMebzeXZycpKdnp4iI9z4tfvJeEpGyvQMzxK+jesQmpaNUSp0xh+CsyBauoD76ImMMcs/fsI9ksuj3BpyYitLfHwpl0f5gBz2msv1sNsQB+vdcTLDPT6vbU14//Nykc4uZQYmbSKrNRjrxERsiF/D+L3brFmk1hTphrhII+L7Eb5Ih28cpa3wEcnuAIsFknMv4wiJyST4tHfxjvnN2w/vL85f7VFx8qk1iZpEN1hYh7t5nv/4MPYzrDFXhJpWkNreOvycYofeb17iA2+j83bQ3oLD21STyf+vtkV6Z81Dkd5tiOMnan9I1GSyv9l4b4+lbNUfPJF52PKyK+PDq7q63kKXYjdayBy2G9eKSf/7KLcO9BWyX7zgHbPnXn5qTY6/tRXEfwiT8PaQs5cXaqIWpz9EDc9FTCaT3oW7hMr2W+MdKfWki3H6dTTPdj9LMq92OiqrYtJ/AQAA///S8LvxSQUAAA==
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
    content: H4sIAAAAAAAA/wAAAP//bFRhb9s2EP1s/opXVZ6TbZLmZJ9aOFhRdFsHLBnmDsPQFAEtnmJmEqkcT26NJP99oCQ7cVZ+EcF79+7e8VEvXxQr64qVDmsVSJCREttQEN20i/TIaCF8N/0nmzbZ1FxdTX+9mv5+NV0eK/pCJc6QFNK0BdO1DUKchS5Q1njT1RSy9G5P9ZDX/jrBydk3c6VshRcofdNoZ5BtcHPbB3BWGNoUrqvr15A1OQVQufZIltsg1OBiiaEQa7HeodTOeUHLviQyOWY3t7Mdb1ZbR/hteXE+xEPwDBvAdNtZJoNVJ4jZle+cgXfY+o4R+kojVdMFwYpgXRBd12SgA1rNAl/FBmNDttHXhJap1WNXQagNOf6oSQcCU0UM8T2+C8RotOt0jcozDIm2kTdW4K6M+SH2MvBZdz10ta+zosoz7RTH+Gcr64Op5Hiz8iwxdnCcxGl+sYK5qqxS42BFD9DlX8t3+PMpPs8TpcrGLJIYe+udo1KQMebzeXZycpKdnp4iI9z4tfvJeEpGyvQMzxK+jesQmpaNUSp0xh+CsyBauoD76ImMMcs/fsI9ksuj3BpyYitLfHwpl0f5gBz2msv1sNsQB+vdcTLDPT6vbU14//Nykc4uZQYmbSKrNRjrxERsiF/D+L3brFmk1hTphrhII+L7Eb5Ih28cpa3wEcnuAIsFknMv4wiJyST4tHfxjvnN2w/vL85f7VFx8qk1iZpEN1hYh7t5nv/4MPYzrDFXhJpWkNreOvycYofeb17iA2+j83bQ3oLD21STyf+vtkV6Z81Dkd5tiOMnan9I1GSyv9l4b4+lbNUfPJF52PKyK+PDq7q63kKXYjdayBy2G9eKSf/7KLcO9BWyX7zgHbPnXn5qTY6/tRXEfwiT8PaQs5cXaqIWpz9EDc9FTCaT3oW7hMr2W+MdKfWki3H6dTTPdj9LMq92OiqrYtJ/AQAA///S8LvxSQUAAA==
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
