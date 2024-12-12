#!/usr/bin/env node
'use strict';
// https://airtable.com/developers/web/api/get-base-schema
import {
  cacheData,
  getData,
  createRecord,
  updateRecord,
  getListId,
  getTables
} from './airtable.js';
import { getTitle } from './linkcopier_actions.js';
import { notify } from './airtable_actions.js';

const testURL = (url) => {
  try {
    return !!new URL(url);
  } catch {
    return !1;
  }
};

const ask = async (input = '') => {
  input = input.trim();
  const tables = await getTables();
  let { url, title, table, category, tags, notes, list, id, done } =
    process.env;
  done = done ? done * 1 : 0;
  if (!url) {
    input = (input || process.env.input || '').replace(/^- /, '');
    let m;
    if (((m = input.match(/\[(.+)\]\(([^()]+)\)$/)), m)) {
      (title = m[1]), (url = m[2]);
    } else {
      url = testURL(input) ? input : undefined;
    }
  }
  url = url?.replace(
    /chrome-extension:\/\/.+?\/suspended\.html#ttl=.+?uri=/,
    ''
  );
  let items = [];
  if (!url) {
    items = [
      {
        title: 'Save Link to Airtable',
        subtitle: input || process.env.input,
        valid: !1
      }
    ];
  } else if (!table) {
    items = Object.keys(tables)
      .filter((t) => t != 'Lists')
      .map((name) => {
        return {
          title: name,
          subtitle: title || url,
          icon: { path: 'media/table.png' },
          arg: ' ',
          variables: { url, title, table: name, mode: 'save' },
          mods: {
            alt: {
              subtitle: 'Edit record',
              arg: ' ',
              icon: { path: 'media/edit.png' },
              variables: { url, title, table: name, mode: 'ask' }
            }
          }
        };
      });
  } else {
    if (testURL(input)) {
      items.push({
        title: `Edit URL: ${input}`,
        subtitle: `Current: ${url}`,
        arg: ' ',
        autocomplete: url,
        quicklookurl: url,
        valid: input != url,
        icon: { path: 'media/link.png' },
        variables: {
          url: input,
          title,
          table,
          category,
          tags,
          notes,
          list,
          done
        }
      });
    } else {
      items.push({
        title: url,
        autocomplete: url,
        quicklookurl: url,
        valid: !1,
        icon: { path: 'media/link.png' }
      });
    }
    if (input || title) {
      items.push({
        title: `Edit Title: '${input || title}'`,
        subtitle: `Current: '${title || ''}'`,
        arg: ' ',
        autocomplete: title,
        valid: input != title,
        icon: { path: 'media/title.png' },
        variables: {
          url,
          title: input,
          table,
          category,
          tags,
          notes,
          list,
          done
        }
      });
    }
    let reg = new RegExp('(^| )' + input.slice(1), 'i');
    if (input.match(/^#.*$/)) {
      tags = tags ? tags.split(',') : [];
      let matches = [];
      tables[table].fields
        .find((f) => f.name == 'Tags')
        .options.choices.forEach(({ name }) => {
          if (reg.test(name) && !tags.includes(name)) {
            matches.push(name);
            items.push({
              title: name,
              arg: ' ',
              icon: { path: 'media/tag.png' },
              // variables: {tags: [...tags, name].join(',')}
              variables: {
                url,
                title,
                table,
                category,
                tags: [...tags, name].join(','),
                notes,
                list,
                done
              }
            });
          }
        });
      if (
        !matches.length &&
        input.length > 1 &&
        !tags.includes(input.slice(1))
      ) {
        items.push({
          title: input.slice(1),
          arg: ' ',
          icon: { path: 'media/tag-new.png' },
          // variables: {tags: [...tags, input.slice(1)].join(',')}
          variables: {
            url,
            title,
            table,
            category,
            tags: [...tags, input.slice(1)].join(','),
            notes,
            list,
            done
          }
        });
      }
      tags = tags.join(',');
    } else if (!category && input.match(/^\/.*$/)) {
      tables[table].fields
        .find((f) => f.name == 'Category')
        .options.choices.forEach(({ name }) => {
          if (reg.test(name))
            items.push({
              title: name,
              arg: ' ',
              icon: { path: 'media/category.png' },
              // variables: {category: name}
              variables: {
                url,
                title,
                table,
                category: name,
                tags,
                notes,
                done,
                list
              }
            });
        });
    } else if (table == 'Links' && input.match(/^\+.*$/) && !list) {
      let { Lists } = await getData();
      Lists.sort(
        (a, b) => new Date(b['Last Modified']) - new Date(a['Last Modified'])
      ).forEach((l) => {
        if (reg.test(l.Name))
          items.push({
            title: l.Name,
            arg: ' ',
            icon: { path: 'media/list.png' },
            variables: {
              url,
              title,
              table,
              category,
              tags,
              notes,
              done,
              list: l.Name
            }
          });
      });
      if (!items.length && input.length > 1) {
        items.push({
          title: input.slice(1),
          arg: ' ',
          icon: { path: 'media/list-new.png' },
          variables: {
            url,
            title,
            table,
            category,
            tags,
            notes,
            done,
            list: input.slice(1)
          }
        });
      }
    }
    if (done) {
      items.push({
        title: 'Completed',
        subtitle: 'Undo complete 􀂒',
        arg: ' ',
        icon: { path: 'media/checked.png' },
        variables: { url, title, table, category, tags, notes, list, done: 0 }
      });
    } else if (input == '.d') {
      items.push({
        title: 'Mark as Completed 􀃲',
        arg: ' ',
        icon: { path: 'media/checked.png' },
        variables: { url, title, table, category, tags, notes, list, done: 1 }
      });
    }
    if (!notes && input) {
      items.push({
        title: 'Add a Note',
        subtitle: input,
        arg: ' ',
        icon: { path: 'media/notes.png' },
        variables: { url, title, table, category, tags, notes: input, list }
      });
    }
    items.unshift({
      title: `Save to ${table}${category ? ` as 􀈭 ${category}` : ''}${
        tags
          ? ` with ${tags
              .split(',')
              .map((t) => '􀆃 ' + t)
              .join(', ')}`
          : ''
      }${done ? ' 􀃲' : ''}${notes ? ` with Note 􀓕 ` : ''}${list ? ', Add to 􀈕 ' + list : ''}`,
      subtitle: title || url,
      arg: ' ',
      icon: { path: 'media/save.png' },
      quicklookurl: url,
      variables: {
        url,
        title,
        table,
        category,
        tags,
        notes,
        list,
        done,
        id,
        mode: 'save'
      },
      mods: { cmd: { subtitle: url, valid: !1 } }
    });
  }
  process.stdout.write(JSON.stringify({ items, variables: { mode: 'ask' } }));
};

/**
 *
 * @param {string} Name
 * @param {string} URL
 * @param {object} params - {List, Note, Done, Tags, Category}
 * @param {string} table - table name
 */
const addLink = async (Name, URL, params = {}, table = 'Links') => {
  if (!Name || !URL) throw new Error('Name and URL are required');
  const tables = await getTables();
  const defaultParams = {
    Note: undefined,
    Done: false,
    Tags: undefined,
    Category: undefined
  };
  let fields = { Name, URL };
  if (params.List && tables[table].fields.find((f) => f.name == 'Lists'))
    fields.Lists = [await getListId(params.List)];
  Object.entries(defaultParams).forEach(([k, v]) => {
    if (params[k] !== undefined || v) fields[k] = params[k] || v;
  });
  await createRecord(tables[table].id, { fields });
};

const editLink = async (id, params = {}, table = 'Links') => {
  const tables = await getTables();
  let fields = {};
  if (params.List && tables[table].fields.find((f) => f.name == 'Lists'))
    fields.Lists = [await getListId(params.List)];
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined) fields[k] = v;
  });
  if (Object.keys(fields).length == 0) throw new Error('No fields to update');
  await updateRecord(tables[table].id, { id, fields });
};

const save = async () => {
  let { url, title, table, category, tags, notes, list, done, id } =
    process.env;
  if (!url) throw new Error('No url provided');
  let message = '';
  try {
    if (id) {
      await editLink(
        id,
        {
          Name: title,
          URL: url,
          Category: category,
          Tags: tags ? tags.split(',') : undefined,
          List: list,
          Done: done == 1,
          Note: notes
        },
        table
      );
      message = `Updated link`;
    } else {
      if (!title) title = (await getTitle(url)).title;
      await addLink(
        title,
        url,
        {
          Category: category,
          Tags: tags ? tags.split(',') : undefined,
          List: list,
          Done: done == 1,
          Note: notes
        },
        table
      );
      message = `Saved ${title} to ${table}`;
    }
  } catch (e) {
    message = 'Error: ' + e.message;
    console.error(e);
  }
  notify(message);
  cacheData();
};

switch (process.env.mode) {
  case 'save':
    save();
    break;
  case 'ask':
  default:
    ask(process.argv[2]);
    break;
}
