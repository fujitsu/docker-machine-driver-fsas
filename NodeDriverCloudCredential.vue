<template>
  <div>
    <h3>Custom Driver Credential</h3>
    <FormField name="apiEndpoint" label="API Endpoint" :rules="required" />
    <FormField name="accessKey" label="Access Key" type="password" :rules="required" />
    <FormField name="secretKey" label="Secret Key" type="password" />
  </div>
</template>

<script>
export default {
  name: 'NodeDriverCloudCredential',
  data() {
    return {
      required: 'This field is required'
    };
  }
};
</script>
