// Serialize writes so a slow response cannot restore a previously selected locale.
// A failed write stays pending until a retry/reconnect; it never spins on failure.
export function createNativeLocaleSync(send: (locale: string) => Promise<void>, onError: (error: unknown) => void) {
  let desired: string | undefined;
  let revision = 0;
  let acknowledged = 0;
  let inFlight = false;
  let disposed = false;

  const flush = async () => {
    if (disposed || inFlight || desired === undefined || acknowledged === revision) return;
    inFlight = true;
    const sendingRevision = revision;
    const sendingLocale = desired;
    try {
      await send(sendingLocale);
      acknowledged = sendingRevision;
    } catch (error) {
      if (!disposed) onError(error);
    } finally {
      inFlight = false;
      if (revision !== sendingRevision) void flush();
    }
  };

  return {
    setLocale(locale: string) {
      if (desired === locale) return;
      desired = locale;
      revision++;
      void flush();
    },
    resync() { revision++; void flush(); },
    retry() { void flush(); },
    dispose() { disposed = true; },
  };
}
