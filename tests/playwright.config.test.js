// @ts-check
const assert = require('node:assert/strict');
const test = require('node:test');

function loadConfigWithEnv(env = {}) {
  const configPath = require.resolve('./playwright.config.js');
  const previousEnv = {};

  for (const [key, value] of Object.entries(env)) {
    previousEnv[key] = process.env[key];
    if (value === undefined) {
      delete process.env[key];
    } else {
      process.env[key] = value;
    }
  }

  try {
    delete require.cache[configPath];
    return require(configPath);
  } finally {
    delete require.cache[configPath];
    for (const [key, value] of Object.entries(previousEnv)) {
      if (value === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = value;
      }
    }
  }
}

function projectByName(config, name) {
  const project = config.projects.find((candidate) => candidate.name === name);
  assert.ok(project, `expected ${name} project config`);
  return project;
}

function uiProjectPort(config) {
  const baseURL = projectByName(config, 'ui-chromium').use.baseURL;
  assert.ok(baseURL, 'expected UI project baseURL');
  return Number(new URL(baseURL).port);
}

function uiWebServer(config) {
  const uiPort = uiProjectPort(config);
  const uiServer = config.webServer.find((server) => server.port === uiPort);
  assert.ok(uiServer, 'expected UI web server config');
  return uiServer;
}

function uiWebServerCommand(config) {
  return uiWebServer(config).command;
}

test('UI E2E uses real UI server by default', () => {
  const config = loadConfigWithEnv({ UI_E2E_DEMO: undefined });

  assert.match(uiWebServerCommand(config), /cmd\/nylas\/main\.go ui --no-browser --port 7363/);
});

test('UI E2E can run against deterministic demo UI', () => {
  const config = loadConfigWithEnv({ UI_E2E_DEMO: 'true' });

  assert.match(uiWebServerCommand(config), /cmd\/nylas\/main\.go demo ui --no-browser --port 7363/);
});

test('UI E2E demo mode does not reuse a non-demo server', () => {
  const config = loadConfigWithEnv({ UI_E2E_DEMO: 'true', CI: undefined });

  assert.equal(uiWebServer(config).reuseExistingServer, false);
});
