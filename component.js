// component.js for Rancher 2.13.1 Custom Node Driver
// Host this file publicly (GitHub Pages, static server) and set as Custom UI URL

import Vue from 'vue';
import { MCI } from '@rancher/monitoring-crd';
import { HARNESS } from '@rancher/shell';
import NodeDriverCloudCredential from './NodeDriverCloudCredential.vue';
import NodeDriverMachineConfig from './NodeDriverMachineConfig.vue';

const ICON_SVG = `
<svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect width="48" height="48" rx="8" fill="#2563EB"/>
  <path d="M12 16h24v16H12z" fill="#60A5FA"/>
  <text x="24" y="28" font-family="sans-serif" font-size="16" fill="white" text-anchor="middle">🚀</text>
</svg>
`.trim();

export default {
  install(Vue) {
    // Register components for your node driver name (replace 'my-custom-driver')
    Vue.component('fsas-cloud-credential', NodeDriverCloudCredential);
    Vue.component('fsas-machine-config', NodeDriverMachineConfig);

    // Override node driver metadata with custom icon
    const originalDriver = window.RANCHER_NODE_DRIVER || {};
    
    window.RANCHER_NODE_DRIVER = {
      ...originalDriver,
      name: 'fsas',
      displayName: 'My Custom Driver',
      icon: `data:image/svg+xml;base64,${btoa(ICON_SVG)}`,
      iconSvg: ICON_SVG,
      active: true
    };
  }
};

// Auto-register when script loads
if (typeof window !== 'undefined') {
  window.addEventListener('load', () => {
    Vue.use(require('./component.js').default);
  });
}
