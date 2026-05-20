import { connect, JSONCodec } from './vendor/nats.ws.js';

export class LocalNATS {
    constructor(url = 'ws://127.0.0.1:9222') {
        this.url = url;
        this.codec = JSONCodec();
        this.connection = null;
        this.connectPromise = null;
        this.reconnectDelay = 500;
    }

    connect() {
        if (!this.connectPromise) {
            this.connectPromise = this.open();
        }
        return this.connectPromise;
    }

    async open() {
        for (;;) {
            try {
                this.connection = await connect({
                    servers: this.url,
                    name: 'Palette Web UI'
                });
                return this.connection;
            } catch (err) {
                console.warn('Local NATS connect failed:', err);
                await delay(this.reconnectDelay);
                this.reconnectDelay = Math.min(5000, this.reconnectDelay * 1.5);
            }
        }
    }

    async subscribe(subject, callback) {
        const nc = await this.connect();
        const sub = nc.subscribe(subject);
        (async () => {
            for await (const msg of sub) {
                try {
                    callback(this.codec.decode(msg.data));
                } catch (err) {
                    console.error('NATS subscription callback failed:', err);
                }
            }
        })();
        return sub;
    }

    async publish(subject, payload = {}) {
        const nc = await this.connect();
        nc.publish(subject, this.codec.encode(payload));
    }

    async request(subject, payload = {}, timeoutMs = 2000) {
        const nc = await this.connect();
        const msg = await nc.request(subject, this.codec.encode(payload), { timeout: timeoutMs });
        return this.codec.decode(msg.data);
    }
}

function delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}
