const canvas = document.getElementById('canvas');
const ctx = canvas.getContext('2d');
const image = document.getElementById('image');
const type = document.getElementById('type');
const note = document.getElementById('annotation-note');
const list = document.getElementById('annotations');
const status = document.getElementById('status');
const annotations = [];
let start = null;

const actual = new Image();
actual.onload = () => {
  canvas.width = actual.naturalWidth;
  canvas.height = actual.naturalHeight;
  ctx.drawImage(actual, 0, 0);
};
actual.src = '/image/actual.png';

function point(event) {
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

function redraw() {
  if (!actual.complete) return;
  ctx.drawImage(actual, 0, 0);
  ctx.strokeStyle = '#ef4444';
  ctx.fillStyle = '#ef4444';
  annotations.forEach((annotation) => {
    if (annotation.type === 'point') {
      ctx.beginPath();
      ctx.arc(annotation.x, annotation.y, 5, 0, Math.PI * 2);
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

function refresh() {
  list.replaceChildren();
  annotations.forEach((annotation, index) => {
    const item = document.createElement('li');
    item.textContent = `${index + 1}. ${annotation.image} ${annotation.type}` +
      ` @ ${annotation.x},${annotation.y}` +
      (annotation.width ? `, ${annotation.width}x${annotation.height}` : '') +
      (annotation.note ? ` — ${annotation.note}` : '');
    list.appendChild(item);
  });
  redraw();
}

canvas.addEventListener('pointerdown', (event) => {
  start = point(event);
  canvas.setPointerCapture(event.pointerId);
});

canvas.addEventListener('pointerup', (event) => {
  if (!start) return;
  const end = point(event);
  const annotation = {
    image: image.value,
    type: type.value,
    x: start.x,
    y: start.y,
    note: note.value,
  };

  if (type.value === 'rectangle') {
    annotation.x = Math.min(start.x, end.x);
    annotation.y = Math.min(start.y, end.y);
    annotation.width = Math.abs(end.x - start.x) + 1;
    annotation.height = Math.abs(end.y - start.y) + 1;
    if (annotation.width < 2 || annotation.height < 2) {
      start = null;
      return;
    }
  }

  annotations.push(annotation);
  start = null;
  note.value = '';
  refresh();
});

async function send(decision) {
  const notes = document.getElementById('notes').value;
  if (decision === 'approved' && (annotations.length || notes.trim())) {
    status.className = 'error';
    status.textContent = 'Remove notes and annotations before approving.';
    return;
  }
  if (decision === 'submitted' && !annotations.length && !notes.trim()) {
    status.className = 'error';
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
    status.className = 'success';
    status.textContent = decision === 'approved'
      ? 'Approved. You may close this page.'
      : 'Feedback saved. You may close this page.';
    document.getElementById('submit').disabled = true;
    document.getElementById('approve').disabled = true;
  } catch (error) {
    status.className = 'error';
    status.textContent = error.message;
  }
}

document.getElementById('submit').onclick = () => send('submitted');
document.getElementById('approve').onclick = () => send('approved');
