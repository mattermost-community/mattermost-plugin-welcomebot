// A client-only stable key for list rendering - never sent to the server.
export function generateKey(): string {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID();
    }
    return `key-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
