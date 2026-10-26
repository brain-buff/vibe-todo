const $ = (s) => document.querySelector(s);

const form = $('#todoForm');
const title = $('#fTitle');
const desc = $('#fDesc');
const due = $('#fDue');
const priority = $('#fPriority');
const list = $('#list');
const empty = $('#empty');
const counter = $('#counter');
const clearDoneBtn = $('#clearDone');
const submitBtn = $('#submitBtn');
const cancelEditBtn = $('#cancelEdit');

// auth elements
const authModal = $('#authModal');
const authForm = $('#authForm');
const authTitle = $('#authTitle');
const authSubmit = $('#authSubmit');
const authCancel = $('#authCancel');
const loginBtn = $('#loginBtn');
const registerBtn = $('#registerBtn');
const logoutBtn = $('#logoutBtn');
const userEmail = $('#userEmail');

let todos = [];
let editingId = null;
let authMode = 'login'; // 'login' | 'register'

const TOKEN_KEY = 'todos_token';
const USER_KEY = 'todos_user';

const PRIORITY = {
  1: { label: 'низкий', cls: 'p1' },
  2: { label: 'средний', cls: 'p2' },
  3: { label: 'высокий', cls: 'p3' },
};

/* ---------- auth ---------- */

function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

function getUser() {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null');
  } catch {
    return null;
  }
}

function setAuth(token, user) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

function clearAuth() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

function isAuthed() {
  return Boolean(getToken());
}

function openAuthModal(mode) {
  authMode = mode;
  authTitle.textContent = mode === 'login' ? 'Вход' : 'Регистрация';
  authSubmit.textContent = mode === 'login' ? 'Войти' : 'Зарегистрироваться';
  authModal.hidden = false;
  $('#aEmail').focus();
}

function closeAuthModal() {
  authModal.hidden = true;
  authForm.reset();
}

function renderAuthUI() {
  const authed = isAuthed();
  const u = getUser();

  loginBtn.hidden = authed;
  registerBtn.hidden = authed;
  logoutBtn.hidden = !authed;
  userEmail.hidden = !authed;
  userEmail.textContent = u ? u.email : '';

  form.hidden = !authed;
  list.hidden = !authed;
  empty.hidden = !authed;
  clearDoneBtn.hidden = !authed || todos.filter((t) => t.done).length === 0;
}

loginBtn.addEventListener('click', () => openAuthModal('login'));
registerBtn.addEventListener('click', () => openAuthModal('register'));
authCancel.addEventListener('click', closeAuthModal);

authModal.addEventListener('click', (e) => {
  if (e.target === authModal) closeAuthModal();
});

authForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const email = $('#aEmail').value.trim();
  const password = $('#aPassword').value;
  if (!email || !password) return;

  const url = authMode === 'login' ? '/api/auth/login' : '/api/auth/register';
  try {
    const data = await req(url, {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    setAuth(data.token, data.user);
    closeAuthModal();
    renderAuthUI();
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

logoutBtn.addEventListener('click', () => {
  clearAuth();
  todos = [];
  render();
  renderAuthUI();
});

/* ---------- api ---------- */

async function req(url, options = {}) {
  const headers = { 'Content-Type': 'application/json' };
  const token = getToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(url, { headers, ...options });
  if (res.status === 401 && isAuthed()) {
    clearAuth();
    renderAuthUI();
  }
  if (!res.ok) {
    let msg = `ошибка ${res.status}`;
    try {
      msg = (await res.json()).error || msg;
    } catch (_) { /* нет тела ответа */ }
    throw new Error(msg);
  }
  return res.json();
}

const api = {
  list: () => req('/api/todos'),
  create: (d) => req('/api/todos', { method: 'POST', body: JSON.stringify(d) }),
  update: (id, d) => req(`/api/todos/${id}`, { method: 'PUT', body: JSON.stringify(d) }),
  remove: (id) => req(`/api/todos/${id}`, { method: 'DELETE' }),
  setDone: (id, done) => req(`/api/todos/${id}/done`, { method: 'PATCH', body: JSON.stringify({ done }) }),
  clearDone: () => req('/api/todos/done', { method: 'DELETE' }),
  addSubtask: (id, t) => req(`/api/todos/${id}/subtasks`, { method: 'POST', body: JSON.stringify({ title: t }) }),
  setSubtaskDone: (id, sid, done) => req(`/api/todos/${id}/subtasks/${sid}/done`, { method: 'PATCH', body: JSON.stringify({ done }) }),
  removeSubtask: (id, sid) => req(`/api/todos/${id}/subtasks/${sid}`, { method: 'DELETE' }),
};

/* ---------- helpers ---------- */

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, (ch) => ({
    '&': '\u0026amp;', '<': '\u0026lt;', '>': '\u0026gt;', '"': '\u0026quot;', "'": '\u0026#39;',
  }[ch]));
}

function plural(n, one, few, many) {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
}

function toLocalInput(iso) {
  const d = new Date(iso);
  const pad = (v) => String(v).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function fmtDue(iso) {
  const d = new Date(iso);
  const now = new Date();
  const startOf = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate());
  const days = Math.round((startOf(d) - startOf(now)) / 864e5);
  const hm = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  const when = `${d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })}, ${hm}`;
  if (days < 0) return { text: `просрочено · ${when}`, cls: 'overdue' };
  if (days === 0) return { text: `сегодня, ${hm}`, cls: 'today' };
  if (days === 1) return { text: `завтра, ${hm}`, cls: 'soon' };
  return { text: when, cls: '' };
}

/* ---------- render ---------- */

function subtaskHTML(s) {
  return `
    <li class="subtask ${s.done ? 'done' : ''}">
      <button class="check check-small" data-action="sub-done" data-id="${s.id}" title="выполнено">✓</button>
      <span>${esc(s.title)}</span>
      <button class="icon-btn sub-del" data-action="sub-del" data-id="${s.id}" title="удалить">✕</button>
    </li>`;
}

function todoHTML(t) {
  const p = PRIORITY[t.priority] || PRIORITY[2];
  const subs = t.subtasks || [];
  const dueEl = t.due_at
    ? (() => { const f = fmtDue(t.due_at); return `<span class="due ${f.cls}">${f.text}</span>`; })()
    : '';
  return `
  <li class="todo${t.done ? ' done' : ''} ${p.cls}" data-id="${t.id}">
    <div class="todo-main">
      <button class="check" data-action="done" title="выполнено">✓</button>
      <div class="todo-body">
        <div class="todo-top">
          <span class="title">${esc(t.title)}</span>
          <span class="badge ${p.cls}">${p.label}</span>
        </div>
        ${t.description ? `<p class="desc">${esc(t.description)}</p>` : ''}
        <div class="todo-meta">
          ${dueEl}
          <div class="actions">
            <button class="icon-btn" data-action="edit" title="редактировать">✎</button>
            <button class="icon-btn danger" data-action="del" title="удалить">✕</button>
          </div>
        </div>
        <div class="subtasks">
          ${subs.length ? `<ul class="sub-list">${subs.map(subtaskHTML).join('')}</ul>` : ''}
          <form class="sub-form">
            <input type="text" placeholder="добавить подзадачу…" maxlength="200" required>
            <button type="submit" class="btn">+</button>
          </form>
        </div>
      </div>
    </div>
  </li>`;
}

function render() {
  list.innerHTML = todos.map(todoHTML).join('');
  const doneCount = todos.filter((t) => t.done).length;
  empty.hidden = todos.length > 0 || !isAuthed();
  empty.textContent = 'пока тихо — добавь первую задачу ✨';
  counter.textContent = todos.length
    ? `${todos.length} ${plural(todos.length, 'задача', 'задачи', 'задач')} · ${doneCount} ${plural(doneCount, 'выполнена', 'выполнены', 'выполнено')}`
    : '';
  clearDoneBtn.hidden = doneCount === 0;
}

async function refresh() {
  if (!isAuthed()) {
    todos = [];
    render();
    return;
  }
  todos = await api.list();
  render();
}

/* ---------- форма ---------- */

function resetForm() {
  editingId = null;
  form.reset();
  priority.value = '2';
  submitBtn.textContent = 'Добавить';
  cancelEditBtn.hidden = true;
}

function startEdit(t) {
  editingId = t.id;
  title.value = t.title;
  desc.value = t.description;
  priority.value = String(t.priority);
  due.value = t.due_at ? toLocalInput(t.due_at) : '';
  submitBtn.textContent = 'Сохранить';
  cancelEditBtn.hidden = false;
  title.focus();
}

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  if (!title.value.trim()) return;
  const data = {
    title: title.value.trim(),
    description: desc.value.trim(),
    priority: Number(priority.value) || 2,
    due_at: due.value ? new Date(due.value).toISOString() : null,
  };
  try {
    if (editingId) await api.update(editingId, data);
    else await api.create(data);
    resetForm();
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

cancelEditBtn.addEventListener('click', resetForm);

clearDoneBtn.addEventListener('click', async () => {
  if (!confirm('Удалить все выполненные задачи?')) return;
  try {
    await api.clearDone();
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

/* ---------- действия в списке (делегирование) ---------- */

list.addEventListener('click', async (e) => {
  const btn = e.target.closest('button[data-action]');
  if (!btn) return;
  const todoEl = btn.closest('.todo');
  const id = Number(todoEl.dataset.id);
  const action = btn.dataset.action;
  const sid = Number(btn.dataset.id);

  try {
    switch (action) {
      case 'done': {
        const t = todos.find((x) => x.id === id);
        await api.setDone(id, !t.done);
        break;
      }
      case 'edit': {
        const t = todos.find((x) => x.id === id);
        startEdit(t);
        return;
      }
      case 'del':
        if (!confirm('Удалить задачу?')) return;
        await api.remove(id);
        break;
      case 'sub-done': {
        const sub = btn.closest('.subtask');
        await api.setSubtaskDone(id, sid, !sub.classList.contains('done'));
        break;
      }
      case 'sub-del':
        if (!confirm('Удалить подзадачу?')) return;
        await api.removeSubtask(id, sid);
        break;
      default:
        return;
    }
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

list.addEventListener('submit', async (e) => {
  const f = e.target.closest('.sub-form');
  if (!f) return;
  e.preventDefault();
  const input = f.querySelector('input');
  const val = input.value.trim();
  if (!val) return;
  const id = Number(f.closest('.todo').dataset.id);
  try {
    await api.addSubtask(id, val);
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

/* ---------- init ---------- */

resetForm();
renderAuthUI();
refresh().catch((err) => alert(err.message));