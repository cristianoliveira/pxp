const decisionSubmitted = 'submitted';
const decisionApproved = 'approved';
const annotationPoint = 'point';
const annotationRectangle = 'rectangle';
const errorClass = 'error';
const successClass = 'success';
const annotationColor = '#ef4444';
const pointRadius = 5;
const fullCircleRadians = Math.PI * 2;
const rectangleSizeOffset = 1;
const minimumRectangleSize = 2;

const canvas = document.getElementById('canvas');
const ctx = canvas.getContext('2d');
const imageSelect = document.getElementById('image');
const annotationTypeSelect = document.getElementById('type');
const annotationNote = document.getElementById('annotation-note');
const annotationList = document.getElementById('annotations');
const status = document.getElementById('status');
const annotations = [];
let startPoint = null;

const actualImage = new Image();
actualImage.onload = () => {
  canvas.width = actualImage.naturalWidth;
  canvas.height = actualImage.naturalHeight;
  ctx.drawImage(actualImage, 0, 0);
};
actualImage.src = '/image/actual.png';

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
  if (!actualImage.complete) return;
  ctx.drawImage(actualImage, 0, 0);
  ctx.strokeStyle = annotationColor;
  ctx.fillStyle = annotationColor;
  annotations.forEach((annotation) => {
    if (annotation.type === annotationPoint) {
      ctx.beginPath();
      ctx.arc(annotation.x, annotation.y, pointRadius, 0, fullCircleRadians);
      ctx.fill();
    } else {
      ctx.strokeRect(
        annotation.x,
        annotation.y,
        annotation.width,
        annotation.height,
      );
    }
  });
}

function renderAnnotations() {
  annotationList.replaceChildren();
  annotations.forEach((annotation, index) => {
    const item = document.createElement('li');
    item.textContent = `${index + 1}. ${annotation.image} ${annotation.type}` +
      ` @ ${annotation.x},${annotation.y}` +
      (annotation.width ? `, ${annotation.width}x${annotation.height}` : '') +
      (annotation.note ? ` — ${annotation.note}` : '');
    annotationList.appendChild(item);
  });
  redrawCanvas();
}

canvas.addEventListener('pointerdown', (event) => {
  startPoint = canvasPointFromEvent(event);
  canvas.setPointerCapture(event.pointerId);
});

canvas.addEventListener('pointerup', (event) => {
  if (!startPoint) return;
  const endPoint = canvasPointFromEvent(event);
  const annotation = {
    image: imageSelect.value,
    type: annotationTypeSelect.value,
    x: startPoint.x,
    y: startPoint.y,
    note: annotationNote.value,
  };

  if (annotationTypeSelect.value === annotationRectangle) {
    annotation.x = Math.min(startPoint.x, endPoint.x);
    annotation.y = Math.min(startPoint.y, endPoint.y);
    annotation.width = Math.abs(endPoint.x - startPoint.x) + rectangleSizeOffset;
    annotation.height = Math.abs(endPoint.y - startPoint.y) + rectangleSizeOffset;
    if (annotation.width < minimumRectangleSize || annotation.height < minimumRectangleSize) {
      startPoint = null;
      return;
    }
  }

  annotations.push(annotation);
  startPoint = null;
  annotationNote.value = '';
  renderAnnotations();
});

async function submitDecision(decision) {
  const notes = document.getElementById('notes').value;
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
      body: JSON.stringify({annotations, notes, decision}),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'feedback was rejected');
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

document.getElementById('submit').onclick = () => submitDecision(decisionSubmitted);
document.getElementById('approve').onclick = () => submitDecision(decisionApproved);
