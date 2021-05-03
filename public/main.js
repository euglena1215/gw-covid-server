document.addEventListener('DOMContentLoaded', () => {
    let loc = window.location;
    let uri = 'ws:';
    if (loc.protocol === 'https:') {
        uri = 'wss:';
    }
    uri += '//' + loc.host;
    uri += loc.pathname + 'ws';

    const ws = new WebSocket(uri)
    ws.onopen = function() {
        console.log('Connected')
    }

    ws.onmessage = function(evt) {
        let out = document.getElementById('output');
        out.innerHTML += evt.data + '<br>';
    }

    const btn = document.querySelector('.btn')
    btn.addEventListener('click', () => {
        ws.send(document.getElementById('input').value)
    })
});
