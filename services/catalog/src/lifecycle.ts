/** Shared shutdown flag: readiness turns false first so Kubernetes stops sending traffic. */
export const lifecycle = { shuttingDown: false };
