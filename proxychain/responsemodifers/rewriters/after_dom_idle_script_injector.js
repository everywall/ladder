(() => {
    document.addEventListener('DOMContentLoaded', (event) => {
        initIdleMutationObserver();
    });
    
    function initIdleMutationObserver() {
        let debounceTimer;
        const debounceDelay = 500; // adjust the delay as needed
    
        const observer = new MutationObserver((mutations) => {
            // Clear the previous timer and set a new one
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => {
                execute();
                observer.disconnect(); // Disconnect after first execution
            }, debounceDelay);
        });
    
        const config = { attributes: false, childList: true, subtree: true };
        observer.observe(document.body, config);
    }
    
    function execute() {
        'SCRIPT_CONTENT_PARAM'
        //console.log('DOM is now idle. Executing...');
    }
})();