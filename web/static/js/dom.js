const E = (id) => document.getElementById(id);
const focusableSelector = "a[href], button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex='-1'])";
let activeModal = null;
let inlineNoticeTimer = null;
let adminPopoverHideTimer = null;
