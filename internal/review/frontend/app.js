const decisionSubmitted = 'submitted';
const decisionApproved = 'approved';
const annotationPoint = 'point';
const annotationRectangle = 'rectangle';
const errorClass = 'error';
const successClass = 'success';
const annotationColor = '#ef4444';
const selectedAnnotationColor = '#f59e0b';
const pointRadius = 5;
const selectedPointRadius = 7;
const fullCircleRadians = Math.PI * 2;
const rectangleSizeOffset = 1;
const minimumRectangleSize = 2;
const keyboardStep = 1;
const keyboardJump = 10;
const cursorSize = 8;
const draftStoragePrefix = 'pxp.review.draft.';

const viewLabels = Object.freeze({
  reference: 'Reference',
  actual: 'Current',
  overlay: 'Overlay',
});
const viewImagePaths = Object.freeze({
  reference: '/image/reference.png',
  actual: '/image/actual.png',
  overlay: '/image/overlay.png',
});
const validAnnotationTypes = new Set([annotationPoint, annotationRectangle]);

const canvas = document.getElementById('canvas');
const ctx = canvas.getContext('2d');
const viewButtons = document.querySelectorAll('[data-view]');
const viewStatus = document.getElementById('view-status');
const interactionStatus = document.getElementById('interaction-status');
const annotationTypeSelect = document.getElementById('type');
const annotationNote = document.getElementById('annotation-note');
const annotationList = document.getElementById('annotations');
const notesInput = document.getElementById('notes');
const status = document.getElementById('status');
const annotations = [];
const viewImages = new Map();
let activeView = 'actual';
let selectedAnnotationId = '';
let startPoint = null;
let keyboardPoint = null;
let keyboardRectangleStart = null;
let editingAnnotationId = '';
let inlineAnnotationId = '';
let editingOriginalNote = '';
let editingTriggerId = '';
let nextDraftID = 1;
let draftStorageKey = '';
let viewLoadVersion = 0;

function imageForView(view) {
  if (!viewImages.has(view)) {
    viewImages.set(view, new Promise((resolve, reject) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => reject(new Error(`Unable to load ${viewLabels[view]} image.`));
      image.src = viewImagePaths[view];
    }));
  }
  return viewImages.get(view);
}

function annotationKey(annotation, index) {
  return annotation.id || `draft-${index}`;
}

function sourceLabel(source) {
  return viewLabels[source] || source;
}

function announceKeyboardPoint(message) {
  interactionStatus.textContent = message;
}

function updateViewControls() {
  viewButtons.forEach((button) => {
    const isActive = button.dataset.view === activeView;
    button.setAttribute('aria-pressed', String(isActive));
  });
  viewStatus.textContent = `Viewing ${sourceLabel(activeView)}`;
  canvas.setAttribute('aria-label', `${sourceLabel(activeView)} screenshot annotation canvas`);
  announceKeyboardPoint(`Use Arrow keys to move on the ${sourceLabel(activeView)} view.`);
}

function clampPoint(point) {
  return {
    x: Math.max(0, Math.min(canvas.width - 1, point.x)),
    y: Math.max(0, Math.min(canvas.height - 1, point.y)),
  };
}

async function setView(view) {
  if (!viewLabels[view]) return;
  if (activeView !== view) keyboardRectangleStart = null;
  activeView = view;
  updateViewControls();
  saveDraft();

  const requestVersion = ++viewLoadVersion;
  try {
    const image = await imageForView(view);
    if (requestVersion !== viewLoadVersion) return;
    canvas.width = image.naturalWidth;
    canvas.height = image.naturalHeight;
    keyboardPoint = clampPoint(keyboardPoint || {
      x: Math.floor((canvas.width - 1) / 2),
      y: Math.floor((canvas.height - 1) / 2),
    });
    redrawCanvas();
  } catch (error) {
    status.className = errorClass;
    status.textContent = error.message;
  }
}

function canvasPointFromEvent(event) {
  const rect = canvas.getBoundingClientRect();
  const style = getComputedStyle(canvas);
  const left = parseFloat(style.borderLeftWidth) || 0;
  const top = parseFloat(style.borderTopWidth) || 0;
  const right = parseFloat(style.borderRightWidth) || 0;
  const bottom = parseFloat(style.borderBottomWidth) || 0;
  const contentWidth = rect.width - left - right;
  const contentHeight = rect.height - top - bottom;

  return {
    x: Math.max(
      0,
      Math.min(
        canvas.width - 1,
        Math.floor((event.clientX - rect.left - left) * canvas.width / contentWidth),
      ),
    ),
    y: Math.max(
      0,
      Math.min(
        canvas.height - 1,
        Math.floor((event.clientY - rect.top - top) * canvas.height / contentHeight),
      ),
    ),
  };
}

function redrawCanvas() {
  const view = activeView;
  const imagePromise = viewImages.get(view);
  if (!imagePromise) return;
  imagePromise.then((image) => {
    if (view !== activeView) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(image, 0, 0);
    annotations
      .filter((annotation) => annotation.image === view)
      .forEach((annotation) => drawAnnotation(annotation));
    drawKeyboardCursor();
  });
}

function drawKeyboardCursor() {
  if (document.activeElement !== canvas || !keyboardPoint) return;
  ctx.strokeStyle = selectedAnnotationColor;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(keyboardPoint.x - cursorSize, keyboardPoint.y);
  ctx.lineTo(keyboardPoint.x + cursorSize, keyboardPoint.y);
  ctx.moveTo(keyboardPoint.x, keyboardPoint.y - cursorSize);
  ctx.lineTo(keyboardPoint.x, keyboardPoint.y + cursorSize);
  ctx.stroke();
}

function drawAnnotation(annotation) {
  const index = annotations.indexOf(annotation);
  const selected = annotationKey(annotation, index) === selectedAnnotationId;
  ctx.strokeStyle = selected ? selectedAnnotationColor : annotationColor;
  ctx.fillStyle = selected ? selectedAnnotationColor : annotationColor;
  ctx.lineWidth = selected ? 3 : 2;
  if (annotation.type === annotationPoint) {
    ctx.beginPath();
    ctx.arc(
      annotation.x,
      annotation.y,
      selected ? selectedPointRadius : pointRadius,
      0,
      fullCircleRadians,
    );
    ctx.fill();
    return;
  }
  ctx.strokeRect(annotation.x, annotation.y, annotation.width, annotation.height);
}

function focusAnnotationAction(annotationId, action) {
  const actionButton = Array.from(
    annotationList.querySelectorAll(`[data-annotation-${action}]`),
  ).find((button) => button.dataset.annotationId === annotationId);
  if (actionButton) actionButton.focus();
  else canvas.focus();
}

function renderAnnotations() {
  annotationList.replaceChildren();
  annotations.slice().reverse().forEach((annotation, displayIndex) => {
    const item = document.createElement('li');
    const selectButton = document.createElement('button');
    const editButton = document.createElement('button');
    const removeButton = document.createElement('button');
    const originalIndex = annotations.indexOf(annotation);
    const key = annotationKey(annotation, originalIndex);
    const description = `${displayIndex + 1}. ${sourceLabel(annotation.image)} ${annotation.type}` +
      ` @ ${annotation.x},${annotation.y}` +
      (annotation.width ? `, ${annotation.width}x${annotation.height}` : '') +
      (annotation.note ? ` — ${annotation.note}` : '');

    selectButton.type = 'button';
    selectButton.dataset.annotationId = key;
    selectButton.dataset.annotationSelect = '';
    selectButton.setAttribute('aria-current', String(key === selectedAnnotationId));
    selectButton.textContent = description;
    selectButton.addEventListener('click', () => selectAnnotation(annotation, key));

    editButton.type = 'button';
    editButton.dataset.annotationId = key;
    editButton.dataset.annotationEdit = '';
    editButton.setAttribute('aria-label', `Edit annotation ${displayIndex + 1}`);
    editButton.textContent = 'Edit note';
    editButton.addEventListener('click', () => beginAnnotationEdit(annotation, key));

    removeButton.type = 'button';
    removeButton.dataset.annotationId = key;
    removeButton.dataset.annotationRemove = '';
    removeButton.setAttribute('aria-label', `Remove annotation ${displayIndex + 1}`);
    removeButton.textContent = 'Remove';
    removeButton.addEventListener('click', () => removeAnnotation(annotation, key));

    item.append(selectButton, editButton, removeButton);
    annotationList.appendChild(item);
  });
  redrawCanvas();
}

function selectAnnotation(annotation, key) {
  if (editingAnnotationId && editingAnnotationId !== key) finishAnnotationEdit(true, false);
  selectedAnnotationId = key;
  renderAnnotations();
  void setView(annotation.image).then(() => canvas.focus());
}

function beginAnnotationEdit(annotation, key) {
  if (inlineAnnotationId) finishInlineAnnotationEdit(true, false);
  if (editingAnnotationId && editingAnnotationId !== key) finishAnnotationEdit(true, false);
  editingAnnotationId = key;
  editingOriginalNote = annotation.note || '';
  editingTriggerId = key;
  selectedAnnotationId = key;
  annotationNote.value = editingOriginalNote;
  renderAnnotations();
  void setView(annotation.image).then(() => {
    annotationNote.focus();
    announceKeyboardPoint(`Editing annotation ${annotations.indexOf(annotation) + 1}. Press Enter to save or Escape to cancel.`);
  });
}

function beginInlineAnnotationEdit(annotation) {
  inlineAnnotationId = annotation.id;
  annotationNote.value = annotation.note || '';
  annotationNote.focus();
  announceKeyboardPoint('Annotation placed. Describe it now; press Enter to save or Escape to leave it blank.');
}

function finishInlineAnnotationEdit(save, restoreFocus) {
  if (!inlineAnnotationId) return;
  const annotation = annotations.find((item) => item.id === inlineAnnotationId);
  const annotationId = inlineAnnotationId;
  if (annotation && !save) annotation.note = '';
  inlineAnnotationId = '';
  annotationNote.value = '';
  saveDraft();
  renderAnnotations();
  if (restoreFocus) focusAnnotationAction(annotationId, 'edit');
  else canvas.focus();
  announceKeyboardPoint(save ? 'Annotation note saved.' : 'Annotation note left blank.');
}

function finishAnnotationEdit(save, restoreFocus) {
  if (!editingAnnotationId) return;
  const annotation = annotations.find((item, index) => annotationKey(item, index) === editingAnnotationId);
  const triggerId = editingTriggerId;
  if (annotation && !save) annotation.note = editingOriginalNote;
  editingAnnotationId = '';
  editingOriginalNote = '';
  editingTriggerId = '';
  annotationNote.value = '';
  saveDraft();
  renderAnnotations();
  if (restoreFocus) focusAnnotationAction(triggerId, 'edit');
}

function removeAnnotation(annotation, key) {
  const index = annotations.indexOf(annotation);
  if (index < 0) return;
  if (editingAnnotationId === key) {
    editingAnnotationId = '';
    editingOriginalNote = '';
    editingTriggerId = '';
    annotationNote.value = '';
  }
  if (inlineAnnotationId === key) {
    inlineAnnotationId = '';
    annotationNote.value = '';
  }
  annotations.splice(index, 1);
  if (selectedAnnotationId === key) selectedAnnotationId = '';
  saveDraft();
  renderAnnotations();
  const nextAnnotation = annotations[index - 1] || annotations[index];
  if (nextAnnotation) {
    focusAnnotationAction(annotationKey(nextAnnotation, annotations.indexOf(nextAnnotation)), 'select');
  } else {
    canvas.focus();
  }
  announceKeyboardPoint('Annotation removed.');
}

function saveDraft() {
  if (!draftStorageKey) return;
  localStorage.setItem(draftStorageKey, JSON.stringify({
    activeView,
    annotationNote: annotationNote.value,
    keyboardRectangleStart,
    annotationType: annotationTypeSelect.value,
    annotations,
    notes: notesInput.value,
  }));
}

function restoreDraft() {
  if (!draftStorageKey) return;
  const rawDraft = localStorage.getItem(draftStorageKey);
  if (!rawDraft) return;

  try {
    const draft = JSON.parse(rawDraft);
    if (!Array.isArray(draft.annotations)) return;
    draft.annotations.forEach((annotation) => {
      if (!viewLabels[annotation.image] || !validAnnotationTypes.has(annotation.type)) return;
      const id = annotation.id || `draft-${nextDraftID++}`;
      const draftNumber = Number.parseInt(id.replace('draft-', ''), 10);
      if (Number.isInteger(draftNumber)) nextDraftID = Math.max(nextDraftID, draftNumber + 1);
      annotations.push({...annotation, id});
    });
    if (typeof draft.notes === 'string') notesInput.value = draft.notes;
    if (typeof draft.annotationNote === 'string') annotationNote.value = draft.annotationNote;
    if (validAnnotationTypes.has(draft.annotationType)) annotationTypeSelect.value = draft.annotationType;
    if (viewLabels[draft.activeView]) activeView = draft.activeView;
    if (draft.keyboardRectangleStart && viewLabels[draft.activeView]) {
      keyboardRectangleStart = draft.keyboardRectangleStart;
    }
  } catch {
    localStorage.removeItem(draftStorageKey);
  }
}

function addAnnotation(start, end) {
  const annotation = {
    id: `draft-${nextDraftID++}`,
    image: activeView,
    type: annotationTypeSelect.value,
    x: start.x,
    y: start.y,
    note: annotationNote.value,
  };

  if (annotation.type === annotationRectangle) {
    annotation.x = Math.min(start.x, end.x);
    annotation.y = Math.min(start.y, end.y);
    annotation.width = Math.abs(end.x - start.x) + rectangleSizeOffset;
    annotation.height = Math.abs(end.y - start.y) + rectangleSizeOffset;
    if (annotation.width < minimumRectangleSize || annotation.height < minimumRectangleSize) {
      announceKeyboardPoint('Rectangle must cover at least two pixels in each direction.');
      return false;
    }
  }

  annotations.push(annotation);
  selectedAnnotationId = annotation.id;
  annotationNote.value = '';
  saveDraft();
  renderAnnotations();
  if (!annotation.note.trim()) beginInlineAnnotationEdit(annotation);
  return true;
}

canvas.addEventListener('pointerdown', (event) => {
  if (inlineAnnotationId) finishInlineAnnotationEdit(true, false);
  if (editingAnnotationId) finishAnnotationEdit(true, false);
  startPoint = canvasPointFromEvent(event);
  keyboardPoint = startPoint;
  canvas.setPointerCapture(event.pointerId);
  redrawCanvas();
});

canvas.addEventListener('pointerup', (event) => {
  if (!startPoint) return;
  const endPoint = canvasPointFromEvent(event);
  keyboardPoint = endPoint;
  addAnnotation(startPoint, endPoint);
  startPoint = null;
});

canvas.addEventListener('pointercancel', () => {
  startPoint = null;
});

function moveKeyboardPoint(deltaX, deltaY) {
  if (!keyboardPoint) return;
  keyboardPoint = clampPoint({
    x: keyboardPoint.x + deltaX,
    y: keyboardPoint.y + deltaY,
  });
  announceKeyboardPoint(
    `${sourceLabel(activeView)} cursor at ${keyboardPoint.x},${keyboardPoint.y}.`,
  );
  redrawCanvas();
}

canvas.addEventListener('keydown', (event) => {
  const step = event.shiftKey ? keyboardJump : keyboardStep;
  if (event.key === 'ArrowUp') {
    event.preventDefault();
    moveKeyboardPoint(0, -step);
    return;
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    moveKeyboardPoint(0, step);
    return;
  }
  if (event.key === 'ArrowLeft') {
    event.preventDefault();
    moveKeyboardPoint(-step, 0);
    return;
  }
  if (event.key === 'ArrowRight') {
    event.preventDefault();
    moveKeyboardPoint(step, 0);
    return;
  }
  if (event.key === 'Escape') {
    event.preventDefault();
    keyboardRectangleStart = null;
    saveDraft();
    announceKeyboardPoint('Rectangle cancelled.');
    redrawCanvas();
    canvas.focus();
    return;
  }
  if (event.key !== 'Enter') return;

  event.preventDefault();
  if (annotationTypeSelect.value === annotationRectangle && !keyboardRectangleStart) {
    keyboardRectangleStart = {...keyboardPoint};
    saveDraft();
    announceKeyboardPoint(
      `Rectangle start set at ${keyboardPoint.x},${keyboardPoint.y}. Move and press Enter to finish.`,
    );
    redrawCanvas();
    return;
  }

  const start = keyboardRectangleStart || keyboardPoint;
  const created = addAnnotation(start, keyboardPoint);
  if (created) keyboardRectangleStart = null;
  saveDraft();
  redrawCanvas();
});

canvas.addEventListener('focus', () => {
  if (!keyboardPoint && canvas.width > 0 && canvas.height > 0) {
    keyboardPoint = clampPoint({
      x: Math.floor((canvas.width - 1) / 2),
      y: Math.floor((canvas.height - 1) / 2),
    });
  }
  announceKeyboardPoint(`Focus is on the ${sourceLabel(activeView)} image.`);
  redrawCanvas();
});

canvas.addEventListener('blur', redrawCanvas);

async function submitDecision(decision) {
  const notes = notesInput.value;
  if (decision === decisionApproved && (annotations.length || notes.trim())) {
    status.className = errorClass;
    status.textContent = 'Remove notes and annotations before approving.';
    return;
  }
  if (decision === decisionSubmitted && !annotations.length && !notes.trim()) {
    status.className = errorClass;
    status.textContent = 'Add a note or annotation before submitting.';
    return;
  }

  status.className = '';
  status.textContent = 'Saving…';
  try {
    const response = await fetch('/api/feedback', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({annotations: annotations.slice().reverse(), notes, decision}),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'feedback was rejected');
    localStorage.removeItem(draftStorageKey);
    status.className = successClass;
    status.textContent = decision === decisionApproved
      ? 'Approved. You may close this page.'
      : 'Feedback saved. You may close this page.';
    document.getElementById('submit').disabled = true;
    document.getElementById('approve').disabled = true;
  } catch (error) {
    status.className = errorClass;
    status.textContent = error.message;
  }
}

async function initialize() {
  try {
    const response = await fetch('/api/session');
    if (!response.ok) throw new Error('Unable to load review session.');
    const session = await response.json();
    draftStorageKey = `${draftStoragePrefix}${session.session_id}`;
    restoreDraft();
    renderAnnotations();
  } catch (error) {
    status.className = errorClass;
    status.textContent = error.message;
  }
  void setView(activeView);
}

viewButtons.forEach((button) => {
  button.addEventListener('click', () => setView(button.dataset.view));
});
annotationTypeSelect.addEventListener('change', saveDraft);
annotationNote.addEventListener('input', () => {
  if (editingAnnotationId) {
    const annotation = annotations.find((item, index) => annotationKey(item, index) === editingAnnotationId);
    if (annotation) annotation.note = annotationNote.value;
  }
  if (inlineAnnotationId) {
    const annotation = annotations.find((item) => item.id === inlineAnnotationId);
    if (annotation) annotation.note = annotationNote.value;
  }
  saveDraft();
});
annotationNote.addEventListener('keydown', (event) => {
  if (event.key === 'Enter') {
    event.preventDefault();
    if (inlineAnnotationId) finishInlineAnnotationEdit(true, true);
    else finishAnnotationEdit(true, true);
  } else if (event.key === 'Escape') {
    event.preventDefault();
    if (inlineAnnotationId) finishInlineAnnotationEdit(false, false);
    else finishAnnotationEdit(false, true);
  }
});
notesInput.addEventListener('input', saveDraft);
document.getElementById('submit').onclick = () => submitDecision(decisionSubmitted);
document.getElementById('approve').onclick = () => submitDecision(decisionApproved);

void initialize();
