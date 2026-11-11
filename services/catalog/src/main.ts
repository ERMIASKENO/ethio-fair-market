import 'reflect-metadata';
import { Logger, LogLevel, LoggerService } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import type { NestExpressApplication } from '@nestjs/platform-express';
import { AppModule } from './app.module';
import { lifecycle } from './lifecycle';
import { observability } from './metrics';

/** JSON logger, same shape as the Go services (time, level, msg, service). */
class JsonLogger implements LoggerService {
  private write(level: string, message: unknown, context?: string) {
    process.stdout.write(JSON.stringify({ time: new Date().toISOString(), level, msg: String(message), context, service: 'catalog' }) + '\n');
  }
  log(m: unknown, c?: string) { this.write('INFO', m, c); }
  error(m: unknown, _t?: string, c?: string) { this.write('ERROR', m, c); }
  warn(m: unknown, c?: string) { this.write('WARN', m, c); }
  debug(m: unknown, c?: string) { if (process.env.LOG_LEVEL === 'debug') this.write('DEBUG', m, c); }
  verbose(m: unknown, c?: string) { this.debug(m, c); }
  setLogLevels(_l: LogLevel[]) {}
}

export async function createApp() {
  const app = await NestFactory.create<NestExpressApplication>(AppModule, { logger: new JsonLogger() });
  app.disable('x-powered-by');
  app.set('trust proxy', true);
  app.use(observability);
  return app;
}

async function bootstrap() {
  const app = await createApp();
  const port = Number(process.env.PORT ?? 8080);
  await app.listen(port, '0.0.0.0');
  Logger.log(`listening on :${port}`, 'Bootstrap');

  // Graceful shutdown: not-ready first, wait for endpoints to update, then close.
  const shutdown = async (signal: string) => {
    if (lifecycle.shuttingDown) return;
    lifecycle.shuttingDown = true;
    Logger.log(`${signal} received, shutting down`, 'Bootstrap');
    await new Promise((r) => setTimeout(r, Number(process.env.SHUTDOWN_DELAY_MS ?? 3000)));
    await app.close();
    process.exit(0);
  };
  process.on('SIGTERM', () => void shutdown('SIGTERM'));
  process.on('SIGINT', () => void shutdown('SIGINT'));
}

if (require.main === module) {
  bootstrap().catch((err) => {
    process.stderr.write(JSON.stringify({ level: 'ERROR', msg: 'fatal', error: String(err) }) + '\n');
    process.exit(1);
  });
}
