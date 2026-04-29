// Navigation modal helpers - extracted from modals_navigation.gohtml inline script
function toggleCcBcc() {
    var ccField = document.getElementById('ccField');
    var bccField = document.getElementById('bccField');
    var toggle = document.getElementById('ccBccToggle');
    ccField.classList.remove('hidden');
    bccField.classList.remove('hidden');
    toggle.classList.add('hidden');
}
