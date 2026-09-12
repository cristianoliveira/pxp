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
const modeButtons = document.querySelectorAll('[data-annotation-mode]');
const annotationNote = document.getElementById('annotation-note');
const annotationList = document.getElementById('annotations');
const notesInput = document.getElementById('notes');
const status = document.getElementById('status');
const annotationEditor = document.getElementById('annotation-editor');
const annotationEditorTitle = document.getElementById('annotation-editor-title');
const annotationEditorContext = document.getElementById('annotation-editor-context');
const annotationEditorStatus = document.getElementById('annotation-editor-status');
const annotationEditorCancel = document.getElementById('annotation-editor-cancel');
const annotationEditorSave = document.getElementById('annotation-editor-save');
const decisionDialog = document.getElementById('decision-dialog');
const decisionSummary = document.getElementById('decision-summary');
const decisionConsequence = document.getElementById('decision-consequence');
const decisionStatus = document.getElementById('decision-status');
const decisionBack = document.getElementById('decision-back');
const decisionConfirm = document.getElementById('decision-confirm');
const contextHeading = document.getElementById('context-heading');
const contextDetails = document.getElementById('implementation-context');
const contextSummary = document.getElementById('context-title');
const contextSections = new Map(
  Array.from(document.querySelectorAll('[data-context-section]'))
    .map((section) => [section.dataset.contextSection, section]),
);
const annotations = [];
const viewImages = new Map();
let activeView = 'actual';
let selectedAnnotationId = '';
let pendingAnnotation = null;
let restoredAnnotationEditor = null;
let startPoint = null;
let keyboardPoint = null;
let keyboardRectangleStart = null;
let nextDraftID = 1;
let draftStorageKey = '';
let viewLoadVersion = 0;
let returnToCanvasAnnouncement = false;
let returnShortcutTimer = 0;
let reviewRound = 1;
let pendingDecision = '';
let decisionInvoker = null;
let decisionSubmitting = false;

function updateContextDisclosureLabel() {
  contextSummary.textContent = contextDetails.open
    ? 'Hide implementation context'
    : 'Show implementation context';
}

function renderImplementationContext(context = {}) {
  contextHeading.textContent = context.title || 'No implementation context was provided for this round.';
  updateContextDisclosureLabel();
  contextSections.forEach((section, field) => {
    const value = typeof context[field] === 'string' ? context[field] : '';
    section.hidden = !value;
    const target = section.querySelector('[data-context-field]');
    if (target) target.textContent = value;
  });
}

contextDetails.addEventListener('toggle', updateContextDisclosureLabel);

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

function isTextEntryTarget(target) {
  return target instanceof HTMLInputElement
    || target instanceof HTMLTextAreaElement
    || target instanceof HTMLSelectElement
    || target?.isContentEditable;
}

function focusCanvas() {
  returnToCanvasAnnouncement = true;
  canvas.focus();
  if (returnToCanvasAnnouncement) {
    returnToCanvasAnnouncement = false;
    announceKeyboardPoint('Focus returned to canvas.');
  }
}

function clearReturnShortcut() {
  window.clearTimeout(returnShortcutTimer);
  returnShortcutTimer = 0;
}

function armReturnShortcut() {
  clearReturnShortcut();
  returnShortcutTimer = window.setTimeout(() => {
    returnShortcutTimer = 0;
  }, 1000);
  announceKeyboardPoint('Press C to return focus to the canvas.');
}

function updateViewControls() {
  viewButtons.forEach((button) => {
    const isActive = button.dataset.view === activeView;
    button.setAttribute('aria-pressed', String(isActive));
  });
  viewStatus.textContent = `Viewing ${sourceLabel(activeView)}`;
  canvas.setAttribute('aria-label', `${sourceLabel(activeView)} screenshot annotation canvas`);
  announceKeyboardPoint(`Viewing ${sourceLabel(activeView)}. Use Arrow keys to move on this view.`);
}

function annotationTypeLabel(type) {
  return type === annotationRectangle ? 'Rectangle' : 'Pin';
}

function updateAnnotationModeControls() {
  modeButtons.forEach((button) => {
    button.setAttribute('aria-pressed', String(button.dataset.annotationMode === annotationTypeSelect.value));
  });
}

function setAnnotationType(type, announce = true) {
  if (!validAnnotationTypes.has(type)) return;
  annotationTypeSelect.value = type;
  keyboardRectangleStart = null;
  updateAnnotationModeControls();
  saveDraft();
  if (announce) announceKeyboardPoint(`${annotationTypeLabel(type)} mode selected.`);
}

function clampPoint(point) {
  return {
    x: Math.max(0, Math.min(canvas.width - 1, point.x)),
    y: Math.max(0, Math.min(canvas.height - 1, point.y)),
  };
}

function pendingViewSwitchMessage() {
  if (startPoint || keyboardRectangleStart) {
    return 'Finish or cancel the rectangle before switching views.';
  }
  return 'Save or cancel the annotation editor before switching views.';
}

async function setView(view, {allowPending = false} = {}) {
  if (!viewLabels[view]) return false;
  if (
    !allowPending
    && (startPoint || keyboardRectangleStart || pendingAnnotation)
  ) {
    announceKeyboardPoint(pendingViewSwitchMessage());
    return false;
  }
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
  return true;
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
    editButton.addEventListener('click', (event) => beginAnnotationEdit(annotation, key, event.currentTarget));

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

function annotationGeometryDescription(annotation) {
  const dimensions = annotation.width
    ? `, ${annotation.width}×${annotation.height}`
    : '';
  return `${sourceLabel(annotation.image)} ${annotationTypeLabel(annotation.type)} at `
    + `${annotation.x},${annotation.y}${dimensions} original pixels.`;
}

function serializePendingAnnotation() {
  if (!pendingAnnotation) return null;
  const value = {
    mode: pendingAnnotation.mode,
    image: pendingAnnotation.image,
    type: pendingAnnotation.type,
    note: annotationNote.value,
  };
  if (pendingAnnotation.mode === 'edit') {
    value.annotationId = pendingAnnotation.key;
    return value;
  }
  return {...value, start: pendingAnnotation.start, end: pendingAnnotation.end};
}

function openAnnotationEditor(pending) {
  if (pendingAnnotation || annotationEditor.open) return;
  pendingAnnotation = pending;
  annotationNote.value = pending.initialNote || pending.annotation?.note || '';
  annotationEditorTitle.textContent = pending.mode === 'edit' ? 'Edit annotation note' : 'Add annotation note';
  const geometry = pending.mode === 'edit'
    ? pending.annotation
    : {
      image: pending.image,
      type: pending.type,
      x: pending.start.x,
      y: pending.start.y,
      width: pending.type === annotationRectangle ? Math.abs(pending.end.x - pending.start.x) + rectangleSizeOffset : 0,
      height: pending.type === annotationRectangle ? Math.abs(pending.end.y - pending.start.y) + rectangleSizeOffset : 0,
    };
  annotationEditorContext.textContent = annotationGeometryDescription(geometry);
  annotationEditorStatus.textContent = '';
  saveDraft();
  annotationEditor.showModal();
  annotationNote.focus();
}

function beginAnnotationEdit(annotation, key, trigger = canvas) {
  selectedAnnotationId = key;
  void setView(annotation.image, {allowPending: true}).then(() => {
    openAnnotationEditor({
      mode: 'edit',
      annotation,
      key,
      image: annotation.image,
      type: annotation.type,
      invoker: trigger,
    });
    announceKeyboardPoint(`Editing annotation ${annotations.indexOf(annotation) + 1}. Press Enter to save or Escape to cancel.`);
  });
}

function beginInlineAnnotationEdit(annotation, initialText = '') {
  openAnnotationEditor({
    mode: 'edit',
    annotation,
    key: annotationKey(annotation, annotations.indexOf(annotation)),
    image: annotation.image,
    type: annotation.type,
    initialNote: `${annotation.note || ''}${initialText}`,
    invoker: canvas,
  });
  announceKeyboardPoint('Editing annotation note. Press Enter to save or Escape to cancel.');
}

function openNewAnnotationEditor(start, end, invoker = canvas) {
  const type = annotationTypeSelect.value;
  if (type === annotationRectangle) {
    const width = Math.abs(end.x - start.x) + rectangleSizeOffset;
    const height = Math.abs(end.y - start.y) + rectangleSizeOffset;
    if (width < minimumRectangleSize || height < minimumRectangleSize) {
      announceKeyboardPoint('Rectangle must cover at least two pixels in each direction.');
      return false;
    }
  }
  openAnnotationEditor({
    mode: 'new',
    image: activeView,
    type,
    start: {...start},
    end: {...end},
    invoker,
  });
  return true;
}

function restoreAnnotationEditorDraft() {
  const draft = restoredAnnotationEditor;
  restoredAnnotationEditor = null;
  if (!draft || pendingAnnotation) return;
  if (draft.mode === 'edit') {
    const index = annotations.findIndex((annotation, itemIndex) => (
      annotationKey(annotation, itemIndex) === draft.annotationId
    ));
    if (index < 0) return;
    const annotation = annotations[index];
    selectedAnnotationId = draft.annotationId;
    void setView(annotation.image, {allowPending: true}).then(() => openAnnotationEditor({
      mode: 'edit', annotation, key: draft.annotationId, image: annotation.image,
      type: annotation.type, initialNote: draft.note, invoker: canvas,
    }));
    return;
  }
  if (
    draft.mode !== 'new'
    || !viewLabels[draft.image]
    || !validAnnotationTypes.has(draft.type)
    || !draft.start || !draft.end
  ) return;
  activeView = draft.image;
  void setView(activeView, {allowPending: true}).then(() => openAnnotationEditor({
    mode: 'new', image: draft.image, type: draft.type,
    start: draft.start, end: draft.end, initialNote: draft.note, invoker: canvas,
  }));
}

function finishAnnotationEditor(save) {
  if (!pendingAnnotation) return;
  const pending = pendingAnnotation;
  const invoker = pending.invoker;
  if (save) {
    if (pending.mode === 'edit') {
      pending.annotation.note = annotationNote.value;
      selectedAnnotationId = pending.key;
    } else {
      const annotation = {
        id: `draft-${nextDraftID++}`,
        image: pending.image,
        type: pending.type,
        x: pending.type === annotationPoint
          ? pending.start.x
          : Math.min(pending.start.x, pending.end.x),
        y: pending.type === annotationPoint
          ? pending.start.y
          : Math.min(pending.start.y, pending.end.y),
        note: annotationNote.value,
      };
      if (pending.type === annotationRectangle) {
        annotation.width = Math.abs(pending.end.x - pending.start.x) + rectangleSizeOffset;
        annotation.height = Math.abs(pending.end.y - pending.start.y) + rectangleSizeOffset;
      }
      annotations.push(annotation);
      selectedAnnotationId = annotation.id;
    }
  }
  annotationEditor.close();
  pendingAnnotation = null;
  annotationNote.value = '';
  annotationEditorStatus.textContent = '';
  saveDraft();
  renderAnnotations();
  if (invoker?.isConnected) invoker.focus();
  else canvas.focus();
  announceKeyboardPoint(
    save ? 'Annotation note saved.'
      : pending.mode === 'new' ? 'Annotation discarded.' : 'Annotation edit cancelled.',
  );
}

function removeAnnotation(annotation, key) {
  const index = annotations.indexOf(annotation);
  if (index < 0) return;
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
    annotationEditor: serializePendingAnnotation(),
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
    if (validAnnotationTypes.has(draft.annotationType)) annotationTypeSelect.value = draft.annotationType;
    if (draft.annotationEditor && typeof draft.annotationEditor === 'object') {
      restoredAnnotationEditor = draft.annotationEditor;
    }
    if (viewLabels[draft.activeView]) activeView = draft.activeView;
    if (draft.keyboardRectangleStart && viewLabels[draft.activeView]) {
      keyboardRectangleStart = draft.keyboardRectangleStart;
    }
  } catch {
    localStorage.removeItem(draftStorageKey);
  }
}

canvas.addEventListener('pointerdown', (event) => {
  startPoint = canvasPointFromEvent(event);
  keyboardPoint = startPoint;
  canvas.setPointerCapture(event.pointerId);
  redrawCanvas();
});

canvas.addEventListener('pointerup', (event) => {
  if (!startPoint) return;
  const endPoint = canvasPointFromEvent(event);
  keyboardPoint = endPoint;
  openNewAnnotationEditor(startPoint, endPoint, canvas);
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
  if (
    event.key.length === 1
    && !event.ctrlKey
    && !event.metaKey
    && !event.altKey
    && !event.shiftKey
    && !event.isComposing
  ) {
    const modeShortcut = event.key.toLowerCase();
    if (modeShortcut === 'p' || modeShortcut === 'r') {
      event.preventDefault();
      setAnnotationType(modeShortcut === 'p' ? annotationPoint : annotationRectangle);
      return;
    }
  }
  if (
    event.key.length === 1
    && event.shiftKey
    && !event.ctrlKey
    && !event.metaKey
    && !event.altKey
    && !event.isComposing
  ) {
    const viewShortcut = {
      r: 'reference',
      c: 'actual',
      o: 'overlay',
    }[event.key.toLowerCase()];
    if (viewShortcut) {
      event.preventDefault();
      void setView(viewShortcut);
      return;
    }
  }
  if (
    event.key.length === 1
    && !event.ctrlKey
    && !event.metaKey
    && !event.altKey
    && !event.isComposing
    && !event.shiftKey
    && !keyboardRectangleStart
  ) {
    const selectedAnnotation = annotations.find((annotation, index) => (
      annotationKey(annotation, index) === selectedAnnotationId
      && annotation.image === activeView
    ));
    if (selectedAnnotation) {
      event.preventDefault();
      beginInlineAnnotationEdit(selectedAnnotation, event.key);
      return;
    }
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
  const opened = openNewAnnotationEditor(start, keyboardPoint, canvas);
  if (opened) keyboardRectangleStart = null;
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
  if (returnToCanvasAnnouncement) {
    returnToCanvasAnnouncement = false;
    announceKeyboardPoint('Focus returned to canvas.');
  } else {
    announceKeyboardPoint(`Focus is on the ${sourceLabel(activeView)} image.`);
  }
  redrawCanvas();
});

canvas.addEventListener('blur', redrawCanvas);

document.addEventListener('keydown', (event) => {
  if (isTextEntryTarget(event.target)) {
    clearReturnShortcut();
    return;
  }
  if (
    event.defaultPrevented
    || event.isComposing
    || event.ctrlKey
    || event.metaKey
    || event.altKey
    || event.shiftKey
  ) return;

  if (event.key.toLowerCase() === 'g') {
    event.preventDefault();
    armReturnShortcut();
    return;
  }

  if (returnShortcutTimer && event.key.toLowerCase() === 'c') {
    event.preventDefault();
    clearReturnShortcut();
    focusCanvas();
    return;
  }

  clearReturnShortcut();
});

document.addEventListener('focusin', clearReturnShortcut);

annotationEditor.addEventListener('cancel', (event) => {
  event.preventDefault();
  finishAnnotationEditor(false);
});
annotationEditor.addEventListener('keydown', (event) => {
  if (event.key !== 'Tab') return;
  const focusable = Array.from(annotationEditor.querySelectorAll('button:not([disabled]), textarea:not([disabled])'));
  if (!focusable.length) return;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
});

decisionDialog.addEventListener('cancel', (event) => {
  event.preventDefault();
  closeDecisionConfirmation();
});
decisionDialog.addEventListener('keydown', (event) => {
  if (event.key !== 'Tab') return;
  const focusable = decisionFocusableElements();
  if (!focusable.length) return;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
});

decisionBack.addEventListener('click', closeDecisionConfirmation);
decisionConfirm.addEventListener('click', () => void confirmDecision());

function validateDecision(decision) {
  const notes = notesInput.value;
  if (decision === decisionApproved && (annotations.length || notes.trim())) {
    status.className = errorClass;
    status.textContent = 'Remove notes and annotations before approving.';
    return false;
  }
  if (decision === decisionSubmitted && !annotations.length && !notes.trim()) {
    status.className = errorClass;
    status.textContent = 'Add a note or annotation before submitting.';
    return false;
  }
  return true;
}

function decisionFocusableElements() {
  return Array.from(decisionDialog.querySelectorAll('button:not([disabled])'));
}

function closeDecisionConfirmation() {
  if (!pendingDecision) return;
  decisionDialog.close();
  const invokingControl = decisionInvoker;
  pendingDecision = '';
  decisionInvoker = null;
  decisionSubmitting = false;
  decisionConfirm.disabled = false;
  decisionBack.disabled = false;
  decisionStatus.className = 'status';
  decisionStatus.textContent = '';
  if (invokingControl && !invokingControl.disabled) invokingControl.focus();
}

function openDecisionConfirmation(decision, invokingControl) {
  if (pendingDecision || decisionSubmitting || !validateDecision(decision)) return;
  pendingDecision = decision;
  decisionInvoker = invokingControl;
  const annotationCount = annotations.length;
  const generalNoteCount = notesInput.value.trim() ? 1 : 0;
  const action = decision === decisionApproved ? 'Approve and finish' : 'Send feedback to agent';
  decisionSummary.textContent = `${action} for Round ${reviewRound}. `
    + `${annotationCount} annotation${annotationCount === 1 ? '' : 's'}, `
    + `${generalNoteCount} general note${generalNoteCount === 1 ? '' : 's'}.`;
  decisionConsequence.textContent = decision === decisionApproved
    ? 'This ends the review. No further feedback will be sent for this round.'
    : 'This sends the saved notes to the agent for another round.';
  decisionConfirm.textContent = action;
  decisionStatus.className = 'status';
  decisionStatus.textContent = '';
  decisionDialog.showModal();
  window.queueMicrotask(() => {
    if (pendingDecision && decisionDialog.open) decisionConfirm.focus();
  });
}

async function submitDecision(decision) {
  const notes = notesInput.value;
  status.className = '';
  status.textContent = 'Saving…';
  decisionStatus.className = 'status';
  decisionStatus.textContent = 'Saving…';
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
    decisionDialog.close();
    pendingDecision = '';
    decisionInvoker = null;
    decisionStatus.className = 'status';
    decisionStatus.textContent = '';
    return true;
  } catch (error) {
    status.className = errorClass;
    status.textContent = error.message;
    decisionStatus.className = `${errorClass} status`;
    decisionStatus.textContent = error.message;
    return false;
  }
}

async function confirmDecision() {
  if (!pendingDecision || decisionSubmitting) return;
  decisionSubmitting = true;
  decisionConfirm.disabled = true;
  decisionBack.disabled = true;
  const persisted = await submitDecision(pendingDecision);
  decisionSubmitting = false;
  if (!persisted && pendingDecision) {
    decisionConfirm.disabled = false;
    decisionBack.disabled = false;
    decisionConfirm.focus();
  }
}

async function initialize() {
  try {
    const response = await fetch('/api/session');
    if (!response.ok) throw new Error('Unable to load review session.');
    const session = await response.json();
    reviewRound = session.round;
    renderImplementationContext(session.context);
    draftStorageKey = `${draftStoragePrefix}${session.session_id}`;
    restoreDraft();
    updateAnnotationModeControls();
    renderAnnotations();
  } catch (error) {
    status.className = errorClass;
    status.textContent = error.message;
  }
  void setView(activeView).then(() => restoreAnnotationEditorDraft());
}

viewButtons.forEach((button) => {
  button.addEventListener('click', () => setView(button.dataset.view));
});
modeButtons.forEach((button) => {
  button.addEventListener('click', () => setAnnotationType(button.dataset.annotationMode));
});
annotationTypeSelect.addEventListener('change', () => setAnnotationType(annotationTypeSelect.value));
annotationNote.addEventListener('input', saveDraft);
annotationNote.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    finishAnnotationEditor(true);
  } else if (event.key === 'Escape') {
    event.preventDefault();
    finishAnnotationEditor(false);
  }
});
notesInput.addEventListener('input', saveDraft);
annotationEditorCancel.addEventListener('click', () => finishAnnotationEditor(false));
annotationEditorSave.addEventListener('click', () => finishAnnotationEditor(true));

document.getElementById('return-to-canvas').onclick = focusCanvas;
document.getElementById('submit').onclick = (event) => openDecisionConfirmation(decisionSubmitted, event.currentTarget);
document.getElementById('approve').onclick = (event) => openDecisionConfirmation(decisionApproved, event.currentTarget);

void initialize();
