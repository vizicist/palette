// Full-screen touch keyboard used for naming presets (Save As / Remove).
// showVirtualKeyboard() returns a Promise that resolves with the entered name
// or rejects with {cancelled: true}.

let keyboardResolve = null;
let keyboardReject = null;
let keyboardCaps = true;
let keyboardValidate = () => '';
let keyboardNormalize = value => value;

export function setupVirtualKeyboard() {
    if (document.getElementById('save-keyboard-modal')) return;

    const modal = document.createElement('div');
    modal.id = 'save-keyboard-modal';
    modal.className = 'keyboard-modal hidden';
    modal.innerHTML = `
        <div class="keyboard-dialog" role="dialog" aria-modal="true" aria-labelledby="keyboard-title">
            <div class="keyboard-title" id="keyboard-title">Save As</div>
            <input id="keyboard-input" class="keyboard-input" type="text" autocomplete="off" spellcheck="false">
            <details id="keyboard-preset-menu" class="keyboard-preset-menu hidden">
                <summary id="keyboard-preset-summary" class="keyboard-preset-summary">Current names</summary>
                <div id="keyboard-preset-options" class="keyboard-preset-options"></div>
            </details>
            <div class="keyboard-error" id="keyboard-error"></div>
            <div class="keyboard-keys" id="keyboard-keys"></div>
            <div class="keyboard-actions">
                <button type="button" class="keyboard-action" data-keyboard-action="cancel">Cancel</button>
                <button type="button" class="keyboard-action primary" data-keyboard-action="submit">Save</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);

    const keys = [
        ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0'],
        ['Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P'],
        ['A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L'],
        ['Z', 'X', 'C', 'V', 'B', 'N', 'M', '-', '_'],
        ['Caps', 'Space', 'Backspace', 'Clear']
    ];
    const keyGrid = modal.querySelector('#keyboard-keys');
    keyGrid.innerHTML = keys.map(row => {
        const buttons = row.map(key => {
            const wide = key === 'Space' || key === 'Backspace' || key === 'Clear' ? ' wide' : '';
            return `<button type="button" class="keyboard-key${wide}" data-key="${key}">${key}</button>`;
        }).join('');
        return `<div class="keyboard-row">${buttons}</div>`;
    }).join('');

    modal.querySelectorAll('.keyboard-key').forEach(button => {
        button.addEventListener('click', () => {
            const input = modal.querySelector('#keyboard-input');
            const key = button.dataset.key;
            if (key === 'Backspace') {
                input.value = input.value.slice(0, -1);
            } else if (key === 'Clear') {
                input.value = '';
            } else if (key === 'Caps') {
                keyboardCaps = !keyboardCaps;
                updateKeyboardCaps();
            } else if (key === 'Space') {
                input.value += ' ';
            } else if (/^[A-Z]$/.test(key)) {
                input.value += keyboardCaps ? key : key.toLowerCase();
            } else {
                input.value += key;
            }
            showKeyboardError('');
            input.focus();
        });
    });

    modal.querySelector('[data-keyboard-action="cancel"]').addEventListener('click', closeVirtualKeyboard);
    modal.querySelector('[data-keyboard-action="submit"]').addEventListener('click', submitVirtualKeyboard);
    modal.querySelector('#keyboard-preset-options').addEventListener('click', event => {
        const button = event.target.closest('.keyboard-preset-option');
        if (!button) return;
        const input = modal.querySelector('#keyboard-input');
        input.value = button.dataset.value || '';
        modal.querySelector('#keyboard-preset-menu').open = false;
        showKeyboardError('');
        input.focus();
    });
    modal.addEventListener('click', event => {
        if (event.target === modal) closeVirtualKeyboard();
    });
    modal.querySelector('#keyboard-input').addEventListener('keydown', event => {
        if (event.key === 'Escape') {
            event.preventDefault();
            closeVirtualKeyboard();
        } else if (event.key === 'Enter') {
            event.preventDefault();
            submitVirtualKeyboard();
        }
    });
    modal.querySelector('#keyboard-input').addEventListener('input', () => showKeyboardError(''));
    updateKeyboardCaps();
}

// showVirtualKeyboard displays the keyboard and resolves with the (normalized)
// entered value. validate(value) should return an error string or '' — it runs
// on submit. normalize(value) cleans the value before resolving.
export function showVirtualKeyboard({ title, initialValue, actionLabel, choices, choiceLabel, validate, normalize }) {
    const modal = document.getElementById('save-keyboard-modal');
    const input = modal.querySelector('#keyboard-input');
    modal.querySelector('#keyboard-title').textContent = title;
    modal.querySelector('[data-keyboard-action="submit"]').textContent = actionLabel || 'Save';
    setKeyboardChoices(modal, choices || [], choiceLabel || 'Current names');
    keyboardValidate = validate || (() => '');
    keyboardNormalize = normalize || (value => value);
    input.value = initialValue || '';
    showKeyboardError('');
    modal.classList.remove('hidden');
    updateKeyboardCaps();
    requestAnimationFrame(() => input.focus());

    return new Promise((resolve, reject) => {
        keyboardResolve = resolve;
        keyboardReject = reject;
    });
}

function setKeyboardChoices(modal, choices, label) {
    const menu = modal.querySelector('#keyboard-preset-menu');
    const summary = modal.querySelector('#keyboard-preset-summary');
    const options = modal.querySelector('#keyboard-preset-options');
    options.replaceChildren();
    menu.open = false;
    if (!choices.length) {
        menu.classList.add('hidden');
        return;
    }

    summary.textContent = label;
    for (const choice of choices) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'keyboard-preset-option';
        button.dataset.value = choice;
        button.textContent = choice;
        options.appendChild(button);
    }
    menu.classList.remove('hidden');
}

function submitVirtualKeyboard() {
    const modal = document.getElementById('save-keyboard-modal');
    const input = modal.querySelector('#keyboard-input');
    const error = keyboardValidate(input.value);
    if (error) {
        showKeyboardError(error);
        input.focus();
        return;
    }
    const cleanName = keyboardNormalize(input.value);
    modal.classList.add('hidden');
    if (keyboardResolve) keyboardResolve(cleanName);
    keyboardResolve = null;
    keyboardReject = null;
}

function closeVirtualKeyboard() {
    const modal = document.getElementById('save-keyboard-modal');
    modal.classList.add('hidden');
    if (keyboardReject) keyboardReject({ cancelled: true });
    keyboardResolve = null;
    keyboardReject = null;
}

function showKeyboardError(message) {
    const error = document.getElementById('keyboard-error');
    if (error) error.textContent = message || '';
}

function updateKeyboardCaps() {
    const modal = document.getElementById('save-keyboard-modal');
    if (!modal) return;
    modal.querySelectorAll('.keyboard-key').forEach(button => {
        const key = button.dataset.key;
        if (/^[A-Z]$/.test(key)) {
            button.textContent = keyboardCaps ? key : key.toLowerCase();
        }
        if (key === 'Caps') {
            button.classList.toggle('active', keyboardCaps);
            button.textContent = keyboardCaps ? 'CAPS' : 'caps';
        }
    });
}
