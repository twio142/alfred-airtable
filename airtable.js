#!/usr/bin/env node
// https://airtable.com/app8pXTEJQGmXMIKC/tblccFizd1lc953it
// https://airtable.com/app8pXTEJQGmXMIKC/api/docs#javascript/introduction
// https://airtable.com/developers/web/api/introduction
'use strict';
import AT from 'airtable';
const { configure, base: _base } = AT;
import { getToken } from './airtable_oauth.js';
import axios from 'axios';
import {
  existsSync,
  mkdirSync,
  statSync,
  readFileSync,
  writeFileSync,
  utimesSync,
  readdirSync
} from 'fs';
import { spawn } from 'child_process';

const CACHE_EXPIRES_IN_MINUTES = process.env.CACHE_EXPIRES_IN_MINUTES || 60;

/**
 * @returns {Promise<Airtable.Base>}
 */
const getBase = async (baseId = process.env.AIRTABLE_BASE_ID) => {
  const accessToken = await getToken();
  configure({
    endpointUrl: 'https://api.airtable.com',
    apiKey: accessToken
  });
  return _base(baseId);
};

/**
 * @param {string} tableId - table id
 * @param {
 *  {string} query - search query
 *  {string[]} tags
 *  {string} formula - filterByFormula; if provided, query and tags will be ignored
 *  {object[]} sort - sort
 * }
 * @returns {Promise<Airtable.Record[]>}
 */
const listRecords = async (
  tableId,
  { query, tags, formula = "{Name}!=''", sort }
) => {
  const base = await getBase();
  if (!formula) {
    formula = ["{Name} != ''"];
    if (query) {
      query = query.toLowerCase();
      formula.push(
        `OR(REGEX_MATCH(LOWER({Name}), "${query}"), REGEX_MATCH(LOWER({Note}), "${query}"))`
      );
    }
    if (tags && tags.length > 0)
      formula.push(
        tags.map(
          (t) =>
            `REGEX_MATCH("," & ARRAYJOIN({Tags}, ",") & ",", ",${t.trim()},")`
        )
      );
    formula = formula.length > 1 ? `AND(${formula.join(',')})` : formula[0];
  }
  return await base(tableId)
    .select({
      view: 'Grid view',
      filterByFormula: formula,
      sort: sort || [{ field: 'Created', direction: 'desc' }]
    })
    .firstPage();
};

/**
 *
 * @param {string} tableId
 * @returns {Promise<Airtable.Record[]>}
 */
const allRecords = async (tableId) => {
  const base = await getBase();
  let output = [];
  await base(tableId)
    .select({ view: 'Grid view' })
    .eachPage((records, fetchNextPage) => {
      output.push(...records);
      fetchNextPage();
    });
  return output;
};

/** Search for list by name, if not found then create one.
 *
 * @param {string} listName
 * @returns {Promise<string>} - list id
 */
const getListId = async (listName) => {
  const { Lists } = await getData();
  let list = Lists.find((l) => l.Name === listName);
  if (list) return list.id;
  const tables = await getTables();
  let records = await createRecord(tables.Lists.id, [
    { fields: { Name: listName } }
  ]);
  if (records?.[0]) return records[0].id;
};

/**
 * @param {string} tableId - table id
 * @param {string} id - record id
 * @returns {Promise<Airtable.Record>}
 */
const getRecord = async (tableId, id) => {
  const base = await getBase();
  return await base(tableId).find(id);
};

/**
 * @param {object[]} newRecords
 * @example newRecords = [{fields: {...}}]
 * @returns {Promise<Airtable.Record[]>}
 */
const createRecord = async (tableId, newRecords) => {
  const base = await getBase();
  newRecords = [].concat(newRecords);
  let rs = [];
  while (newRecords.length)
    rs = rs.concat(
      await base(tableId).create(newRecords.splice(0, 10), { typecast: true })
    );
  cacheData();
  return rs;
};

/**
 * @param newRecords = [{id, fields: {...}}]
 * @returns {Promise<Airtable.Record[]>}
 */
const updateRecord = async (tableId, newRecords) => {
  const base = await getBase();
  newRecords = [].concat(newRecords);
  let rs = [];
  while (newRecords.length) {
    rs.push(
      ...(await base(tableId).update(newRecords.splice(0, 10), {
        typecast: true
      }))
    );
  }
  cacheData();
  return rs;
};

/**
 * @param {string} tableId
 * @param {string[] or string} recordIds
 */
const deleteRecord = async (tableId, recordIds) => {
  const base = await getBase();
  recordIds = [].concat(recordIds);
  let rs = [];
  while (recordIds.length) {
    rs.push(...(await base(tableId).destroy(recordIds.splice(0, 10))));
  }
  cacheData();
  return rs;
};

/**
 *
 * @returns {Promise<object[]>}
 */
const listBases = async () => {
  const accessToken = await getToken();
  let { data } = await axios({
    method: 'GET',
    url: 'https://api.airtable.com/v0/meta/bases',
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  return data.bases;
};

/**
 * Meta data of all tables
 *
 * @param {string} accessToken
 * @param {string} baseId
 * @returns {Promise<object>} - {name: {id, fields: {...}}}
 */
const getTables = async (force = !1) => {
  const cacheDir = process.env.alfred_workflow_cache;
  if (!existsSync(cacheDir)) mkdirSync(cacheDir, { recursive: true });
  const cache = `${cacheDir}/metadata.json`;
  if (
    !force &&
    existsSync(cache) &&
    new Date() - statSync(cache).mtime <= 36e5 * 24
  )
    return JSON.parse(readFileSync(cache));
  const accessToken = await getToken(),
    baseId = process.env.AIRTABLE_BASE_ID;
  let { data } = await axios({
    method: 'GET',
    url: `https://api.airtable.com/v0/meta/bases/${baseId}/tables`,
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  let tables = {};
  data.tables.forEach((t) => (tables[t.name] = t));
  writeFileSync(cache, JSON.stringify(tables));
  return tables;
};

/**
 * Cache data of all tables
 */
const cacheData = async () => {
  const cacheDir = process.env.alfred_workflow_cache;
  if (!existsSync(`${cacheDir}/tables`))
    mkdirSync(`${cacheDir}/tables`, { recursive: true });
  const tables = await getTables(!0);
  for (let t in tables) {
    let records = await allRecords(tables[t].id);
    writeFileSync(
      `${cacheDir}/tables/${t}.json`,
      JSON.stringify(records.map((r) => ({ ...r.fields, id: r.id })))
    );
  }
  utimesSync(`${cacheDir}/tables`, new Date(), new Date());
};

/**
 * Get data of all tables
 * @returns {Promise<object>} - {table: [records]}
 */
const getData = async (force) => {
  const cacheDir = process.env.alfred_workflow_cache;
  if (
    !existsSync(`${cacheDir}/tables`) ||
    !existsSync(`${cacheDir}/metadata.json`)
  )
    await cacheData();
  let tables = readdirSync(`${cacheDir}/tables`).filter(
    (t) => !t.startsWith('.')
  );
  if (!tables.length) {
    await cacheData();
    tables = readdirSync(`${cacheDir}/tables`).filter(
      (t) => !t.startsWith('.')
    );
  }
  if (
    force ||
    statSync(`${cacheDir}/tables`).mtime <
      new Date() - CACHE_EXPIRES_IN_MINUTES * 6e4
  ) {
    spawn(new URL(import.meta.url).pathname, {
      detached: true,
      stdio: 'ignore',
      env: process.env
    }).unref();
  }
  let data = {};
  tables.forEach(
    (t) =>
      (data[t.replace(/\.json$/, '')] = JSON.parse(
        readFileSync(`${cacheDir}/tables/${t}`)
      ))
  );
  return data;
};

if (import.meta.url === `file://${process.argv[1]}`) cacheData();

export {
  listRecords,
  allRecords,
  getListId,
  getRecord,
  createRecord,
  updateRecord,
  deleteRecord,
  listBases,
  getTables,
  cacheData,
  getData
};
