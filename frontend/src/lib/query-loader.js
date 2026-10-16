// Debounce typing, abort superseded requests, and reject late responses even
// when a transport ignores AbortSignal. A disposed page never receives results.
export function createQueryLoader(fetcher, handlers = {}, delay = 250) {
  let timer, controller, revision = 0, lastKey, disposed = false;
  function schedule(key, force = false) {
    if (disposed || (!force && key === lastKey)) return;
    lastKey = key;
    const current = ++revision;
    clearTimeout(timer);
    controller?.abort();
    handlers.loading?.();
    timer = setTimeout(async () => {
      controller = new AbortController();
      try {
        const value = await fetcher(key, controller.signal);
        if (!disposed && revision === current) handlers.success?.(value);
      } catch (error) {
        if (!disposed && revision === current && error.name !== "AbortError") handlers.error?.(error);
      }
    }, delay);
  }
  return { schedule, refresh: () => { if (lastKey !== undefined) schedule(lastKey, true); },
    dispose: () => { disposed = true; revision++; clearTimeout(timer); controller?.abort(); } };
}
