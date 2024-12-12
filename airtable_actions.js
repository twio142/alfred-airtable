#!/usr/bin/env node
'use strict';
import { existsSync, readFileSync, writeFileSync } from 'fs';
import { basename, join } from 'path';
import {
  getListId,
  getRecord,
  updateRecord,
  createRecord,
  deleteRecord,
  getTables,
  getData
} from './airtable.js';
import { execFile } from 'child_process';

/**
 *
 * @param {string} lc - linkcopier file path or content
 * @returns {object} - {Name, Links, Note}
 */
const lc2List = (lc) => {
  let Links = [],
    lines,
    Name;
  if (existsSync(lc)) {
    lines = readFileSync(lc, 'utf8').trim().split(/\n+/);
    Name = basename(lc, '.md');
  } else {
    lines = lc.trim().split(/\n+/);
    Name = lines[0].startsWith('# ')
      ? lines.shift().replace(/^# /, '')
      : undefined;
  }
  let Note = lines
    .filter((l) => l.startsWith('> '))
    .map((l) => l.replace('> ', ''))
    .join('\n');
  lines.forEach((l) => {
    let m;
    if (((m = l.match(/^- \[(.+)]\((.+?)\)$/)), m)) {
      Links.push({
        Name: m[1],
        URL: m[2].replace(
          /chrome-extension:\/\/.+?\/suspended\.html#ttl=.+?uri=/,
          ''
        )
      });
    } else if (((m = /^ {4}> (.+)$/), m && Links.length)) {
      Links[Links.length - 1].Note = m[1];
    }
  });
  return { Name, Links, Note };
};

/**
 *
 * @param {object} list
 * @example list = {Name, Links, Note}
 */
const addList = async (list) => {
  const tables = await getTables();
  let listId;
  if (list.Name) {
    listId = await getListId(list.Name);
    if (list.Note) {
      let Record = await getRecord(tables.Links.id, listId);
      let Note = Record.get('Note');
      Note += (Note ? '\n' : '') + list.Note;
      await updateRecord(tables.Links.id, [{ id: listId, fields: { Note } }]);
    }
  }
  let Links = list.Links.map((l) => {
    l.Lists = [listId];
    l.Done = false;
    return { fields: l };
  });
  await createRecord(tables.Links.id, Links);
};

/**
 *
 * @param {string} listId
 * @param {boolean} deleteLinks - also delete links in the list
 */
const deleteList = async (listId, deleteLinks = !1) => {
  const tables = await getTables();
  if (deleteLinks) {
    const { Links } = await getData();
    let toDelete = Links.filter(
      (l) => l.Lists?.length == 1 && l.Lists?.includes(listId)
    );
    if (toDelete.length)
      deleteRecord(
        tables.Links.id,
        toDelete.map((l) => l.id)
      );
  }
  await deleteRecord(tables.Lists.id, listId);
};

const list2Lc = async (listId) => {
  const { Lists, Links } = await getData();
  let list = Lists.find((l) => l.id == listId);
  if (!list) return;
  let lc = list.Name + '.md';
  let content = list.Note ? '> ' + list.Note + '\n\n' : '';
  content += Links.filter((l) => l.Lists?.includes(listId) && !l.Done)
    .map((l) => `- [${l.Name}](${l.URL})`)
    .join('\n');
  writeFileSync(join(__dirname, 'link_copiers', lc), content);
  execFile('afplay', ['media/arrow.m4a']);
};

const notify = (message) =>
  execFile('terminal-notifier', [
    '-title',
    'Airtable',
    '-message',
    message,
    '-sender',
    'com.runningwithcrayons.Alfred',
    '-contentImage',
    'media/airtable.png'
  ]);

if (import.meta.url === `file://${process.argv[1]}`) {
  let p, m;
  const { mode, tableId, listId, id } = process.env;
  switch (mode) {
    case 'cache':
      getData(!0);
      break;
    case 'list2lc':
      list2Lc(listId);
      break;
    case 'complete':
      p = () => updateRecord(tableId, { id, fields: { Done: !0 } });
      m = 'Marked as completed';
      break;
    case 'delete-link':
      p = () => deleteRecord(tableId, id);
      m = 'Link deleted';
      break;
    case 'delete-links':
    case 'delete-list':
      p = () => deleteList(id, mode == 'delete-links');
      m = 'List deleted';
      break;
  }
  if (p)
    p()
      .then(() => notify(m))
      .catch((e) => notify('Error: ' + e.message));
}

export { lc2List, addList, deleteList, list2Lc, notify };
