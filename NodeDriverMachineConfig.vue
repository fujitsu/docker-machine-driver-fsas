<template>
  <div>
    <h3>Machine Configuration</h3>
    <FormField name="region" label="Region" />
    <FormField name="instanceType" label="Instance Type" />
    <FormField name="sshUser" label="SSH User" />
    <LabeledInput type="checkbox" name="userData" label="User Data" />
  </div>
</template>

<script>
export default {
  name: 'NodeDriverMachineConfig'
};
</script>
