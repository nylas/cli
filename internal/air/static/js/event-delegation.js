// Centralized click delegation. Modules register handlers under a
// data-action key; this single document-level listener dispatches.
var Actions = (function() {
    var handlers = Object.create(null);
    return {
        register: function(name, handler) {
            handlers[name] = handler;
        },
        dispatch: function(target, event) {
            var handler = handlers[target.dataset.action];
            if (handler) handler(target, event);
        },
        has: function(name) {
            return name in handlers;
        }
    };
})();

document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    Actions.dispatch(target, e);
});

document.addEventListener('change', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    Actions.dispatch(target, e);
});

document.addEventListener('input', function(e) {
    var target = e.target.closest('[data-action-input]');
    if (!target) return;
    var handler = Actions._inputHandlers && Actions._inputHandlers[target.dataset.actionInput];
    if (handler) handler(target, e);
});
