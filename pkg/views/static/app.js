/**
 * s3xplorer Application Scripts
 */

// Theme management
function toggleTheme() {
  const html = document.documentElement;
  const isDark = html.classList.contains('dark');

  if (isDark) {
    html.classList.remove('dark');
    localStorage.setItem('theme', 'light');
  } else {
    html.classList.add('dark');
    localStorage.setItem('theme', 'dark');
  }
}

// Initialize theme from localStorage or system preference
(function initTheme() {
  const saved = localStorage.getItem('theme');
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const theme = saved || (prefersDark ? 'dark' : 'light');

  if (theme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
})();

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
  // Ctrl/Cmd + K: Focus search
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    const searchInput = document.getElementById('searchstr');
    if (searchInput) {
      searchInput.focus();
    }
  }

  // Escape: Clear focus
  if (e.key === 'Escape') {
    const searchInput = document.getElementById('searchstr');
    if (document.activeElement === searchInput) {
      searchInput.blur();
    }
  }
});

// Table row navigation with arrow keys
document.addEventListener('keydown', (e) => {
  if (!['ArrowUp', 'ArrowDown', 'Enter'].includes(e.key)) return;

  const table = document.querySelector('table[role="grid"]');
  if (!table) return;

  const rows = Array.from(table.querySelectorAll('tbody tr'));
  const currentRow = document.activeElement.closest('tr');
  const currentIndex = rows.indexOf(currentRow);

  if (currentIndex === -1) return;

  // Arrow down: Move to next row
  if (e.key === 'ArrowDown' && currentIndex < rows.length - 1) {
    e.preventDefault();
    const nextRow = rows[currentIndex + 1];
    const link = nextRow.querySelector('a');
    if (link) link.focus();
  }

  // Arrow up: Move to previous row
  if (e.key === 'ArrowUp' && currentIndex > 0) {
    e.preventDefault();
    const prevRow = rows[currentIndex - 1];
    const link = prevRow.querySelector('a');
    if (link) link.focus();
  }

  // Enter: Activate focused link
  if (e.key === 'Enter' && document.activeElement.tagName === 'A') {
    e.preventDefault();
    document.activeElement.click();
  }
});

// Home: Focus first table row
// End: Focus last table row
document.addEventListener('keydown', (e) => {
  if (!['Home', 'End'].includes(e.key)) return;

  const table = document.querySelector('table[role="grid"]');
  if (!table) return;

  const rows = Array.from(table.querySelectorAll('tbody tr'));
  if (rows.length === 0) return;

  e.preventDefault();

  if (e.key === 'Home') {
    const firstLink = rows[0].querySelector('a');
    if (firstLink) firstLink.focus();
  } else {
    const lastLink = rows[rows.length - 1].querySelector('a');
    if (lastLink) lastLink.focus();
  }
});

// Toggle all file checkboxes
function toggleAllCheckboxes(checked) {
  const checkboxes = document.querySelectorAll('.file-checkbox');
  checkboxes.forEach(checkbox => {
    checkbox.checked = checked;
  });
  updateDeleteButton();
}

// Update delete button state based on checkbox selection
function updateDeleteButton() {
  const checkboxes = document.querySelectorAll('.file-checkbox:checked');
  const deleteButton = document.getElementById('delete-button');
  const deleteCount = document.getElementById('delete-count');

  if (!deleteButton || !deleteCount) return;

  const count = checkboxes.length;
  deleteCount.textContent = count;
  deleteButton.disabled = count === 0;

  // Update select-all checkbox state
  const selectAll = document.getElementById('select-all');
  if (selectAll) {
    const allCheckboxes = document.querySelectorAll('.file-checkbox');
    selectAll.checked = allCheckboxes.length > 0 && count === allCheckboxes.length;
    selectAll.indeterminate = count > 0 && count < allCheckboxes.length;
  }
}

// Submit delete form with selected file keys
function submitDeleteForm() {
  const checkboxes = document.querySelectorAll('.file-checkbox:checked');
  if (checkboxes.length === 0) return;

  // Confirm deletion
  const fileText = checkboxes.length === 1 ? 'this file' : `these ${checkboxes.length} files`;
  if (!confirm(`Are you sure you want to delete ${fileText}?`)) {
    return;
  }

  // Build form with keys
  const form = document.getElementById('delete-form');
  const container = document.getElementById('delete-keys-container');
  container.innerHTML = ''; // Clear previous

  checkboxes.forEach(checkbox => {
    const input = document.createElement('input');
    input.type = 'hidden';
    input.name = 'keys';
    input.value = checkbox.dataset.key;
    container.appendChild(input);
  });

  form.submit();
}
