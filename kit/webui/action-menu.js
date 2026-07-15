// A small centered popup menu used for preset actions (Move / Rename / Delete)
// and their follow-up choices. showActionMenu() returns a Promise that resolves
// with the chosen item id, or null if the menu was cancelled/dismissed.

let menuResolve = null;

export function setupActionMenu() {
    if (document.getElementById('action-menu-overlay')) return;

    const overlay = document.createElement('div');
    overlay.id = 'action-menu-overlay';
    overlay.className = 'modal-overlay hidden';
    overlay.innerHTML = `
        <div id="action-menu" role="dialog" aria-modal="true" aria-labelledby="action-menu-title">
            <div id="action-menu-title" class="action-menu-title"></div>
            <div id="action-menu-buttons" class="action-menu-buttons"></div>
        </div>
    `;
    document.body.appendChild(overlay);

    overlay.addEventListener('click', event => {
        if (event.target === overlay) closeActionMenu(null);
    });
    overlay.querySelector('#action-menu-buttons').addEventListener('click', event => {
        const button = event.target.closest('[data-menu-id]');
        if (!button) return;
        closeActionMenu(button.dataset.menuId || null);
    });
    document.addEventListener('keydown', event => {
        if (event.key !== 'Escape') return;
        if (!overlay.classList.contains('hidden')) closeActionMenu(null);
    });
}

// showActionMenu displays a titled list of buttons. items is an array of
// { id, label, danger? }. Resolves with the selected id or null.
export function showActionMenu({ title, items, cancelLabel = 'Cancel' }) {
    const overlay = document.getElementById('action-menu-overlay');
    overlay.querySelector('#action-menu-title').textContent = title || '';

    const container = overlay.querySelector('#action-menu-buttons');
    container.replaceChildren();
    for (const item of items) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'modal-button action-menu-btn' + (item.danger ? ' danger' : '');
        button.dataset.menuId = item.id;
        button.textContent = item.label;
        container.appendChild(button);
    }
    const cancel = document.createElement('button');
    cancel.type = 'button';
    cancel.className = 'modal-button action-menu-cancel';
    cancel.dataset.menuId = '';
    cancel.textContent = cancelLabel;
    container.appendChild(cancel);

    overlay.classList.remove('hidden');
    return new Promise(resolve => { menuResolve = resolve; });
}

function closeActionMenu(id) {
    const overlay = document.getElementById('action-menu-overlay');
    overlay.classList.add('hidden');
    const resolve = menuResolve;
    menuResolve = null;
    if (resolve) resolve(id || null);
}
