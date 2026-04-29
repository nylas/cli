// Cache initialization overlay - runs immediately, before any other scripts
(function() {
    var params = new URLSearchParams(window.location.search);
    if (params.get('init') === '1') {
        var overlay = document.getElementById('initOverlay');
        var progressBar = document.getElementById('initProgressBar');
        var statusText = document.getElementById('initStatus');

        // Show overlay immediately with flex display
        overlay.style.display = 'flex';

        var statuses = [
            'Preparing workspace',
            'Clearing old data',
            'Setting up folders',
            'Connecting to API',
            'Loading preferences',
            'Almost ready...'
        ];

        var progress = 0;
        var statusIndex = 0;
        var duration = 3000; // 3 seconds
        var interval = 50;   // Update every 50ms
        var steps = duration / interval;
        var increment = 100 / steps;

        var timer = setInterval(function() {
            progress += increment;
            progressBar.style.width = Math.min(progress, 100) + '%';

            // Update status text at intervals
            var newIndex = Math.floor((progress / 100) * statuses.length);
            if (newIndex !== statusIndex && newIndex < statuses.length) {
                statusIndex = newIndex;
                statusText.textContent = statuses[statusIndex];
            }

            if (progress >= 100) {
                clearInterval(timer);
                statusText.textContent = 'Ready!';

                // Remove init param and refresh
                setTimeout(function() {
                    var newUrl = window.location.origin + window.location.pathname;
                    window.location.replace(newUrl);
                }, 300);
            }
        }, interval);
    }
})();
