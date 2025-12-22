const api = {
    getBooks: () => fetch('/api/books').then(r => r.json()),
    addBook: (name) => fetch('/api/books', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
    }).then(r => r.json()),
    deleteBook: (id) => fetch(`/api/books/${id}`, { method: 'DELETE' }),
    updateBook: (book) => fetch(`/api/books/${book.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(book)
    }).then(r => r.json()),

    getPages: (bookId) => fetch(`/api/pages?bookId=${bookId}`).then(r => r.json()),
    addPage: (page) => fetch('/api/pages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(page)
    }).then(r => r.json()),
    deletePage: (id) => fetch(`/api/pages/${id}`, { method: 'DELETE' }),
    updatePage: (page) => fetch(`/api/pages/${page.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(page)
    }).then(r => r.json())
};

// State
let currentBookId = null;
let editingPageId = null;
let editingBookId = null;
let books = [];

// DOM Elements
const bookListEl = document.getElementById('bookList');
const mainContent = document.getElementById('mainContent');
const emptyState = document.getElementById('emptyState');
const bookView = document.getElementById('bookView');
const currentBookTitle = document.getElementById('currentBookTitle');
const pageCountBadge = document.getElementById('pageCountBadge');
const pagesGrid = document.getElementById('pagesGrid');
const bookSearch = document.getElementById('bookSearch');

// Modals
const bookModal = document.getElementById('bookModal');
const pageModal = document.getElementById('pageModal');
const bookForm = document.getElementById('bookForm');
const pageForm = document.getElementById('pageForm');

// Initialization
async function init() {
    await loadBooks();
    setupEventListeners();
}

// Logic
async function loadBooks() {
    books = await api.getBooks();
    renderBooks(books);
}

function renderBooks(list) {
    bookListEl.innerHTML = list.map(b => `
        <li class="book-item ${currentBookId === b.id ? 'active' : ''}" onclick="selectBook(${b.id})">
            <span>${b.name}</span>
            <ion-icon name="chevron-forward-outline"></ion-icon>
        </li>
    `).join('');
}

window.selectBook = async (id) => {
    currentBookId = id;
    const book = books.find(b => b.id === id);
    if (!book) return;

    renderBooks(books); // Update active state

    // Show View
    emptyState.classList.add('hidden');
    bookView.classList.remove('hidden');

    currentBookTitle.textContent = book.name;

    loadPages(id);
};

async function loadPages(bookId) {
    const pages = await api.getPages(bookId);
    pageCountBadge.textContent = `${pages.length} Pages`;

    pagesGrid.innerHTML = pages.map(p => `
        <div class="page-card">
            <div class="page-header">
                <span class="page-number">PAGE ${p.number}</span>
            </div>
            <p class="page-content">${p.content}</p>
            <div class="page-actions">
                <button class="btn btn-small" onclick="editPage(${p.id})">
                    <ion-icon name="create"></ion-icon> Edit
                </button>
                <button class="btn btn-small danger" onclick="deletePage(${p.id})">
                    <ion-icon name="trash"></ion-icon> Delete
                </button>
            </div>
        </div>
    `).join('');
}

window.deletePage = async (id) => {
    if (!confirm('Are you sure you want to delete this page?')) return;
    await api.deletePage(id);
    loadPages(currentBookId);
};

window.editPage = async (id) => {
    const pages = await api.getPages(currentBookId);
    const page = pages.find(p => p.id === id);
    if (!page) return;

    editingPageId = id;

    // Fill form
    const form = pageForm;
    form.querySelector('[name="number"]').value = page.number;
    form.querySelector('[name="content"]').value = page.content;

    // Update modal title/button
    pageModal.querySelector('.modal-header h3').textContent = 'Edit Page';
    pageModal.querySelector('.modal-footer .btn.primary').textContent = 'Save Changes';

    showModal(pageModal);
};

// Event Listeners
function setupEventListeners() {
    // Buttons
    document.getElementById('addBookBtn').onclick = () => {
        editingBookId = null;
        bookForm.reset();
        bookModal.querySelector('.modal-header h3').textContent = 'New Book';
        bookModal.querySelector('.modal-footer .btn.primary').textContent = 'Create Book';
        showModal(bookModal);
    };

    document.getElementById('editBookBtn').onclick = () => {
        const book = books.find(b => b.id === currentBookId);
        if (!book) return;
        editingBookId = currentBookId;
        bookForm.reset();
        bookForm.querySelector('[name="name"]').value = book.name;
        bookModal.querySelector('.modal-header h3').textContent = 'Edit Book';
        bookModal.querySelector('.modal-footer .btn.primary').textContent = 'Save Changes';
        showModal(bookModal);
    };
    document.getElementById('addPageBtn').onclick = () => {
        editingPageId = null;
        pageForm.reset();
        pageModal.querySelector('.modal-header h3').textContent = 'New Page';
        pageModal.querySelector('.modal-footer .btn.primary').textContent = 'Add Page';
        showModal(pageModal);
    };

    document.getElementById('deleteBookBtn').onclick = async () => {
        if (!confirm('Delete this book and all its pages?')) return;
        await api.deleteBook(currentBookId);
        currentBookId = null;
        emptyState.classList.remove('hidden');
        bookView.classList.add('hidden');
        loadBooks();
    };

    // Close Modals
    document.querySelectorAll('.close-modal').forEach(btn => {
        btn.onclick = () => {
            bookModal.classList.add('hidden');
            pageModal.classList.add('hidden');
        };
    });

    // Forms
    bookForm.onsubmit = async (e) => {
        e.preventDefault();
        const formData = new FormData(bookForm);
        const name = formData.get('name');

        if (editingBookId) {
            await api.updateBook({ id: editingBookId, name });
            // Update local state if needed
            const book = books.find(b => b.id === editingBookId);
            if (book) book.name = name;
            currentBookTitle.textContent = name;
        } else {
            await api.addBook(name);
        }

        bookModal.classList.add('hidden');
        bookForm.reset();
        editingBookId = null;
        loadBooks();
    };

    pageForm.onsubmit = async (e) => {
        e.preventDefault();
        const formData = new FormData(pageForm);
        const data = {
            bookId: currentBookId,
            number: parseInt(formData.get('number')),
            content: formData.get('content')
        };

        if (editingPageId) {
            await api.updatePage({ ...data, id: editingPageId });
        } else {
            await api.addPage(data);
        }

        pageModal.classList.add('hidden');
        pageForm.reset();
        editingPageId = null;
        loadPages(currentBookId);
    };

    // Search
    bookSearch.oninput = (e) => {
        const term = e.target.value.toLowerCase();
        const filtered = books.filter(b => b.name.toLowerCase().includes(term));
        renderBooks(filtered);
    };
}

function showModal(modal) {
    modal.classList.remove('hidden');
}

// Start
init();
