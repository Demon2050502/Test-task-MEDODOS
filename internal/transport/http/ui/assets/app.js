const taskForm = document.getElementById("task-form");
const recurringForm = document.getElementById("recurring-form");
const refreshButton = document.getElementById("refresh-button");
const recurrenceType = document.getElementById("recurrence-type");
const addDateButton = document.getElementById("add-date-button");
const specificDatesList = document.getElementById("specific-dates-list");
const urgentQueue = document.getElementById("urgent-queue");
const regularQueue = document.getElementById("regular-queue");
const recurringList = document.getElementById("recurring-list");
const urgentCount = document.getElementById("urgent-count");
const regularCount = document.getElementById("regular-count");
const toast = document.getElementById("toast");

document.addEventListener("DOMContentLoaded", () => {
  bindEvents();
  syncRecurringFields();
  loadAll();
});

function bindEvents() {
  taskForm.addEventListener("submit", onTaskSubmit);
  recurringForm.addEventListener("submit", onRecurringSubmit);
  recurrenceType.addEventListener("change", syncRecurringFields);
  recurringForm.elements.start_date.addEventListener("change", syncSpecificDateMinimums);
  refreshButton.addEventListener("click", loadAll);
  addDateButton.addEventListener("click", () => addSpecificDateField());

  urgentQueue.addEventListener("click", onTaskAction);
  regularQueue.addEventListener("click", onTaskAction);
  recurringList.addEventListener("click", onRecurringAction);
  specificDatesList.addEventListener("click", onSpecificDateAction);
}

async function loadAll() {
  await Promise.all([loadTasks(), loadRecurringTasks()]);
}

async function loadTasks() {
  const response = await fetchJSON("/api/v1/tasks");
  const urgentTasks = response.urgent_queue || [];
  const regularTasks = response.regular_queue || [];

  urgentCount.textContent = String(urgentTasks.length);
  regularCount.textContent = String(regularTasks.length);

  urgentQueue.innerHTML = renderTaskList(urgentTasks, "В экстренной очереди пока пусто.");
  regularQueue.innerHTML = renderTaskList(regularTasks, "Обычных задач пока нет.");
}

async function loadRecurringTasks() {
  const recurringTasks = await fetchJSON("/api/v1/recurring-tasks");
  recurringList.innerHTML = renderRecurringList(recurringTasks);
}

async function onTaskSubmit(event) {
  event.preventDefault();
  const formData = new FormData(taskForm);
  const payload = {
    title: formData.get("title"),
    description: formData.get("description"),
    priority: formData.get("priority"),
  };

  const status = formData.get("status");
  if (status) {
    payload.status = status;
  }

  await sendJSON("/api/v1/tasks", "POST", payload);
  taskForm.reset();
  showToast("Задача создана.");
  await loadTasks();
}

async function onRecurringSubmit(event) {
  event.preventDefault();
  const formData = new FormData(recurringForm);
  const payload = {
    title: formData.get("title"),
    description: formData.get("description"),
    priority: formData.get("priority"),
    recurrence: {
      type: formData.get("type"),
      start_date: formData.get("start_date"),
    },
  };

  if (payload.recurrence.type === "daily") {
    payload.recurrence.every_n_days = Number(formData.get("every_n_days") || 1);
  }

  if (payload.recurrence.type === "monthly") {
    payload.recurrence.day_of_month = Number(formData.get("day_of_month") || 1);
  }

  if (payload.recurrence.type === "day_parity") {
    payload.recurrence.parity = formData.get("parity");
  }

  if (payload.recurrence.type === "specific_dates") {
    payload.recurrence.dates = getSpecificDates();
    if (!payload.recurrence.dates.length) {
      showToast("Добавь хотя бы одну дату.");
      return;
    }
  }

  await sendJSON("/api/v1/recurring-tasks", "POST", payload);
  recurringForm.reset();
  resetSpecificDates();
  syncRecurringFields();
  showToast("Периодическая задача создана.");
  await loadAll();
}

async function onTaskAction(event) {
  const actionButton = event.target.closest("[data-action]");
  if (!actionButton) {
    return;
  }

  const card = actionButton.closest("[data-task-id]");
  if (!card) {
    return;
  }

  const taskId = card.dataset.taskId;
  const action = actionButton.dataset.action;

  if (action === "delete") {
    await sendJSON(`/api/v1/tasks/${taskId}`, "DELETE");
    showToast("Задача удалена.");
    await loadTasks();
    return;
  }

  if (action === "advance") {
    const task = await fetchJSON(`/api/v1/tasks/${taskId}`);
    const nextStatus = getNextStatus(task.status);
    await sendJSON(`/api/v1/tasks/${taskId}`, "PUT", {
      title: task.title,
      description: task.description,
      status: nextStatus,
      priority: task.priority,
    });
    showToast("Статус задачи обновлен.");
    await loadTasks();
  }
}

async function onRecurringAction(event) {
  const actionButton = event.target.closest("[data-action='delete-recurring']");
  if (!actionButton) {
    return;
  }

  const card = actionButton.closest("[data-recurring-id]");
  if (!card) {
    return;
  }

  await sendJSON(`/api/v1/recurring-tasks/${card.dataset.recurringId}`, "DELETE");
  showToast("Периодическая задача удалена.");
  await loadAll();
}

function renderTaskList(tasks, emptyText) {
  if (!tasks.length) {
    return `<div class="empty-state">${emptyText}</div>`;
  }

  return tasks.map((task) => `
    <article class="task-card ${task.priority === "urgent" ? "urgent" : ""}" data-task-id="${task.id}">
      <div class="card-top">
        <h3 class="card-title">${escapeHTML(task.title)}</h3>
        <div class="pill-row">
          <span class="pill ${task.priority === "urgent" ? "urgent" : ""}">${task.priority === "urgent" ? "Экстренная" : "Обычная"}</span>
          <span class="pill queue">${task.queue === "urgent" ? "Экстренная очередь" : "Обычная очередь"}</span>
        </div>
      </div>
      <p class="card-text">${escapeHTML(task.description || "Без описания")}</p>
      <div class="card-meta">
        <span class="pill">Статус: ${formatStatus(task.status)}</span>
        ${task.scheduled_for ? `<span class="pill">Дата: ${task.scheduled_for}</span>` : ""}
      </div>
      <div class="card-actions">
        ${task.status !== "done" ? `<button class="mini-button" type="button" data-action="advance">Сдвинуть статус</button>` : ""}
        <button class="mini-button danger" type="button" data-action="delete">Удалить</button>
      </div>
    </article>
  `).join("");
}

function renderRecurringList(tasks) {
  if (!tasks.length) {
    return `<div class="empty-state">Периодических задач пока нет.</div>`;
  }

  return tasks.map((task) => `
    <article class="recurring-card" data-recurring-id="${task.id}">
      <div class="card-top">
        <h3 class="card-title">${escapeHTML(task.title)}</h3>
        <span class="pill ${task.priority === "urgent" ? "urgent" : ""}">
          ${task.priority === "urgent" ? "Экстренный шаблон" : "Обычный шаблон"}
        </span>
      </div>
      <p class="card-text">${escapeHTML(task.description || "Без описания")}</p>
      <div class="card-meta">
        <span class="pill">Тип: ${formatRecurrence(task.recurrence)}</span>
        <span class="pill">Старт: ${task.recurrence.start_date}</span>
        ${task.last_generated_for ? `<span class="pill">Последняя генерация: ${task.last_generated_for}</span>` : ""}
      </div>
      <div class="card-actions">
        <button class="mini-button danger" type="button" data-action="delete-recurring">Удалить</button>
      </div>
    </article>
  `).join("");
}

function syncRecurringFields() {
  const type = recurrenceType.value;
  document.querySelectorAll("[data-field]").forEach((node) => {
    node.classList.add("hidden");
  });

  if (type === "daily") {
    document.querySelector("[data-field='every_n_days']").classList.remove("hidden");
  }

  if (type === "monthly") {
    document.querySelector("[data-field='day_of_month']").classList.remove("hidden");
  }

  if (type === "day_parity") {
    document.querySelector("[data-field='parity']").classList.remove("hidden");
  }

  if (type === "specific_dates") {
    document.querySelector("[data-field='dates']").classList.remove("hidden");
    if (!specificDatesList.children.length) {
      addSpecificDateField();
    }
  }

  syncSpecificDateMinimums();
}

function onSpecificDateAction(event) {
  const removeButton = event.target.closest("[data-action='remove-date']");
  if (!removeButton) {
    return;
  }

  removeButton.closest(".date-row")?.remove();
  if (!specificDatesList.children.length) {
    addSpecificDateField();
  }
}

function addSpecificDateField(value = "") {
  const row = document.createElement("div");
  row.className = "date-row";
  row.innerHTML = `
    <input name="specific_date" type="date" value="${escapeAttribute(value)}">
    <button class="mini-button danger" type="button" data-action="remove-date">Убрать</button>
  `;

  specificDatesList.append(row);
  syncSpecificDateMinimums();
}

function resetSpecificDates() {
  specificDatesList.innerHTML = "";
  addSpecificDateField();
}

function syncSpecificDateMinimums() {
  const startDate = recurringForm.elements.start_date.value;
  specificDatesList.querySelectorAll('input[name="specific_date"]').forEach((input) => {
    input.min = startDate || "";
  });
}

function getSpecificDates() {
  return Array.from(specificDatesList.querySelectorAll('input[name="specific_date"]'))
    .map((input) => input.value)
    .filter(Boolean);
}

function getNextStatus(status) {
  if (status === "new") {
    return "in_progress";
  }
  if (status === "in_progress") {
    return "done";
  }
  return "done";
}

function formatStatus(status) {
  return {
    new: "Новая",
    in_progress: "В работе",
    done: "Выполнена",
  }[status] || status;
}

function formatRecurrence(recurrence) {
  if (!recurrence) {
    return "Не указано";
  }

  if (recurrence.type === "daily") {
    return `Каждые ${recurrence.every_n_days || 1} дн.`;
  }
  if (recurrence.type === "monthly") {
    return `Каждый месяц ${recurrence.day_of_month} числа`;
  }
  if (recurrence.type === "specific_dates") {
    return `Конкретные даты: ${(recurrence.dates || []).join(", ")}`;
  }
  if (recurrence.type === "day_parity") {
    return recurrence.parity === "even" ? "Четные дни" : "Нечетные дни";
  }
  return recurrence.type;
}

async function fetchJSON(url) {
  const response = await fetch(url);
  if (!response.ok) {
    await handleError(response);
  }
  return response.json();
}

async function sendJSON(url, method, payload) {
  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
    },
    body: payload ? JSON.stringify(payload) : undefined,
  });

  if (!response.ok) {
    await handleError(response);
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
}

async function handleError(response) {
  let message = "Произошла ошибка.";
  try {
    const payload = await response.json();
    message = payload.error || message;
  } catch (_) {
  }

  showToast(message);
  throw new Error(message);
}

function showToast(message) {
  toast.textContent = message;
  toast.classList.remove("hidden");
  clearTimeout(showToast.timer);
  showToast.timer = setTimeout(() => {
    toast.classList.add("hidden");
  }, 2600);
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function escapeAttribute(value) {
  return escapeHTML(value);
}
