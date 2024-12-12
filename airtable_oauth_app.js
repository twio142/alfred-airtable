#!/usr/bin/env node

import { randomBytes, createHash } from 'crypto';
import { URL } from 'url';
import axios from 'axios';
import { stringify } from 'qs';
import express from 'express';
import { writeFileSync } from 'fs';

const app = express();
// set up environment variables
// if you have not created a .env file by following the README instructions this will not work
import _configs from './.config.js';
const {
  clientId: _clientId,
  clientSecret: _clientSecret,
  port: _port,
  redirectUri: _redirectUri,
  scope: _scope,
  airtableUrl: _airtableUrl
} = _configs;

const clientId = _clientId.trim();
const clientSecret = _clientSecret.trim();
// if you edit the port you will need to edit the redirectUri
const port = _port;
// if you edit the path of this URL will you will need to edit the /airtable-oauth route to match your changes
const redirectUri = _redirectUri.trim();
const scope = _scope.trim();
const airtableUrl = _airtableUrl.trim();

const encodedCredentials = Buffer.from(`${clientId}:${clientSecret}`).toString(
  'base64'
);
const authorizationHeader = `Basic ${encodedCredentials}`;
const authorizationCache = {};

app.get('/', (_, res) => {
  // prevents others from impersonating Airtable
  const state = randomBytes(100).toString('base64url');

  // prevents others from impersonating you
  const codeVerifier = randomBytes(96).toString('base64url'); // 128 characters
  const codeChallengeMethod = 'S256';
  const codeChallenge = createHash('sha256')
    .update(codeVerifier) // hash the code verifier with the sha256 algorithm
    .digest('base64') // base64 encode, needs to be transformed to base64url
    .replace(/=/g, '') // remove =
    .replace(/\+/g, '-') // replace + with -
    .replace(/\//g, '_'); // replace / with _ now base64url encoded

  // ideally, entries in this cache expires after ~10-15 minutes
  authorizationCache[state] = {
    // we'll use this in the redirect url route
    codeVerifier
    // any other data you want to store, like the user's ID
  };

  // build the authorization URL
  const authorizationUrl = new URL(`${airtableUrl}/oauth2/v1/authorize`);
  authorizationUrl.searchParams.set('code_challenge', codeChallenge);
  authorizationUrl.searchParams.set(
    'code_challenge_method',
    codeChallengeMethod
  );
  authorizationUrl.searchParams.set('state', state);
  authorizationUrl.searchParams.set('client_id', clientId);
  authorizationUrl.searchParams.set('redirect_uri', redirectUri);
  authorizationUrl.searchParams.set('response_type', 'code');
  // your OAuth integration register with these scopes in the management page
  authorizationUrl.searchParams.set('scope', scope);

  // redirect the user and request authorization
  res.redirect(authorizationUrl.toString());
});

// route that user is redirected to after successful or failed authorization
// Note that one exemption is that if your client_id is invalid or the provided
// redirect_uri does exactly match what Airtable has stored, the user will not
// be redirected to this route, even with an error.
app.get('/airtable-oauth', (req, res) => {
  const state = req.query.state;
  const cached = authorizationCache[state];
  // validate request, you can include other custom checks here as well
  if (cached === undefined) {
    res.send('This request was not from Airtable!');
    return;
  }
  // clear the cache
  delete authorizationCache[state];

  // Check if the redirect includes an error code.
  // Note that if your client_id and redirect_uri do not match the user will never be re-directed
  // Note also that if you did not include "state" in the request, then this redirect would also not include "state"
  if (req.query.error) {
    const error = req.query.error;
    const errorDescription = req.query.error_description;
    res.send(`
      There was an error authorizing this request.
      <br/>Error: "${error}"
      <br/>Error Description: "${errorDescription}"
    `);
    return;
  }

  // since the authorization didn't error, we know there's a grant code in the query
  // we also retrieve the stashed code_verifier for this request
  const code = req.query.code;
  const codeVerifier = cached.codeVerifier;

  const headers = {
    // Content-Type is always required
    'Content-Type': 'application/x-www-form-urlencoded'
  };
  if (clientSecret !== '') {
    // Authorization is required if your integration has a client secret
    // omit it otherwise
    headers.Authorization = authorizationHeader;
  }

  axios({
    method: 'POST',
    url: `${airtableUrl}/oauth2/v1/token`,
    headers,
    // stringify the request body like a URL query string
    data: stringify({
      // client_id is optional if authorization header provided
      // required otherwise.
      client_id: clientId,
      code_verifier: codeVerifier,
      redirect_uri: redirectUri,
      code,
      grant_type: 'authorization_code'
    })
  })
    .then((response) => {
      let data = response.data;
      data.expires_at = data.expires_in + parseInt(new Date() / 1000);
      data.refresh_expires_at =
        data.refresh_expires_in + parseInt(new Date() / 1000);
      writeFileSync('.credentials.json', JSON.stringify(data));
      console.log(data.access_token);
      res.end('Success! You can close this tab now.');
      process.exit(0);
    })
    .catch((e) => {
      console.error('uh oh, something went wrong', e);
      res.end('Uh oh, something went wrong');
      process.exit(1);
    });
});

app.get('/refresh', (req, res) => {
  const refresh_token = req.query.refresh_token;
  const headers = {
    'Content-Type': 'application/x-www-form-urlencoded'
  };
  if (clientSecret !== '') {
    headers.Authorization = authorizationHeader;
  }
  axios({
    method: 'POST',
    url: `${airtableUrl}/oauth2/v1/token`,
    headers,
    data: stringify({
      client_id: clientId,
      refresh_token,
      scope,
      grant_type: 'refresh_token'
    })
  })
    .then((response) => {
      let data = response.data;
      data.expires_at = data.expires_in + parseInt(new Date() / 1000);
      data.refresh_expires_at =
        data.refresh_expires_in + parseInt(new Date() / 1000);
      writeFileSync('.credentials.json', JSON.stringify(data));
      process.stdout.write(data.access_token);
      res.end(data.access_token);
      process.exit(0);
    })
    .catch((e) => {
      console.error('uh oh, something went wrong', e);
      res.end('Uh oh, something went wrong');
      process.exit(1);
    });
});

app.listen(port, () => {
  // console.log('Server listening on port ' + port);
  new Promise((resolve) => setTimeout(resolve, 2e4)).then(() =>
    process.exit(1)
  );
});
