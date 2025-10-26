import http from 'k6/http';
import {
    check,
    sleep
} from 'k6';

export const options = {
    vus: 100, 
    duration: '30s',
    thresholds: {
        http_req_duration: ['p(90)<200'], 
    },
};

function getRandomCep() {
    const ceps = [
        "01310100",
        "01310-100"
    ]
    return ceps[Math.floor(Math.random() * ceps.length)];
}

export default function () {
    const cep = getRandomCep();
    const url = `http://brasilcep-api:8080/cep/${cep}`;

    const res = http.get(url);

    check(res, {
        'status 200': (r) => r.status === 200,
    });

    sleep(0.1);
}