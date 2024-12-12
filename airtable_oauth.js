import { existsSync, unlinkSync, readFileSync } from 'fs';
import { execFile, execFileSync } from 'child_process';
import { resolve } from 'path';
import Config from './.config.js';
const { port } = Config;

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

const __filename = new URL(import.meta.url).pathname;

const sleepUntil = (f, timeoutMs) => {
  return new Promise((resolve, reject) => {
    const timeWas = new Date();
    const wait = setInterval(() => {
      if (f()) {
        clearInterval(wait);
        resolve();
      } else if (new Date() - timeWas > timeoutMs) {
        clearInterval(wait);
        reject();
      }
    }, 200);
  });
};

const getToken = async () => {
  if (existsSync('.credentials.json')) {
    const { access_token, expires_at, refresh_token, refresh_expires_at } =
      JSON.parse(readFileSync('.credentials.json'));
    if (access_token && expires_at > new Date() / 1000) {
      return access_token;
    } else if (refresh_token && refresh_expires_at > new Date() / 1000) {
      execFile(resolve(__filename, '../airtable_oauth_app.js'));
      await sleep(1e3);
      return execFileSync('curl', [
        '-s',
        '-L',
        `http://localhost:${port}/refresh?refresh_token=${refresh_token}`
      ]).toString();
    } else {
      unlinkSync('.credentials.json');
    }
  }
  execFile(resolve(__filename, '../airtable_oauth_app.js'));
  await sleep(1e3);
  execFile('open', [`http://localhost:${port}`]);
  await sleepUntil(() => existsSync('.credentials.json'), 2e4);
  return JSON.parse(readFileSync('.credentials.json')).access_token;
};

if (import.meta.url === `file://${process.argv[1]}`)
  getToken().then(console.log).catch(console.error);

export { getToken };
