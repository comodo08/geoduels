export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    const { readServerConfig } = await import('./lib/runtime-config.server');
    readServerConfig();
  }
}
