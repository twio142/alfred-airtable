#!/usr/bin/env node
'use strict';
import { getData, getTables } from './airtable.js';
import { pinyin } from 'pinyin-pro';

const match = (title, url) => {
  try {
    url = new URL(url).host
      .split('.')
      .slice(0, -1)
      .filter((x) => !['www', 'm', 'co'].includes(x))
      .join(' ');
  } catch {
    url = '';
  }
  return [title, pinyin(title, { nonZh: 'consecutive' }), url].join(' ');
};

const allLists = async () => {
  let { Lists } = await getData();
  let items = Lists.sort(
    (a, b) => new Date(b['Last Modified']) - new Date(a['Last Modified'])
  )
    .sort((a, b) => (a.Status === 'Done') * 1 - (b.Status == 'Done') * 1)
    .map((l) => {
      let linksCount =
        '􀉣 ' + l['Number of Links to Read'] + '/' + (l.Links?.length || 0);
      let notes = l.Note;
      notes = notes ? '􀓕 ' + notes : '';
      let rUrl = l['Record URL'];
      let subtitle = [linksCount, notes].filter(Boolean).join('  ·  ');
      let largetype =
        (notes ? notes + '\n\n' : '') +
        (l['Link-names']?.map((l) => '- ' + l).join('\n') || '');
      return {
        title: l.Name,
        subtitle,
        text: { largetype, copy: rUrl },
        icon: { path: 'media/list.png' },
        match: match(l.Name) + ' ' + notes,
        arg: ' ',
        action: { text: l.id },
        variables: { id: l.id, rUrl },
        mods: {
          ['alt+shift']: {
            subtitle: 'Open record',
            arg: rUrl,
            variables: { url: rUrl }
          },
          cmd: {
            subtitle: 'Add link to list',
            variables: { list: l.Name, table: 'Links' },
            icon: { path: 'media/link-new.png' }
          },
          shift: {
            subtitle: 'Send to link copier',
            icon: { path: 'media/clip.png' },
            variables: { mode: 'list2lc', listId: l.id }
          },
          ctrl: {
            subtitle: 'Delete list',
            variables: { id: l.id, mode: 'delete-links' },
            icon: { path: 'media/delete.png' }
          },
          'ctrl+alt': {
            subtitle: 'Delete list but keep links in it',
            icon: { path: 'media/delete.png' },
            variables: { id: l.id, mode: 'delete-list' }
          },
          fn: {
            subtitle: 'Rebuild cache',
            arg: '__CACHE__',
            icon: { path: 'media/reload.png' },
            variables: { mode: 'cache' }
          }
        }
      };
    });
  return { items };
};

const lookIntoList = async (listId) => {
  const tables = await getTables();
  const { Links } = await getData();
  let items = Links.filter((l) => (listId ? l.Lists?.includes(listId) : !0))
    .sort((a, b) =>
      a.Done == b.Done
        ? new Date(b['Created']) - new Date(a['Created'])
        : !!a.Done - !!b.Done
    )
    .map((l) => {
      let done = l.Done ? '􀃲 ' : '';
      let tags = (l.Tags || []).map((x) => '􀆃' + x).join(' ');
      let notes = l.Note ? '􀓕 ' + l.Note : '';
      let lists = (l['List-Names'] || []).join(', ');
      lists = lists ? '􀈕 ' + lists : '';
      let category = l.Category ? '􀈭 ' + l.Category : '';
      let subtitle =
        done + [tags, lists, category, notes].filter(Boolean).join('  ·  ');
      let url = l.URL,
        rUrl = l['Record URL'];
      let largetype = [l.Name, '􀉣 ' + url, tags, lists, category, notes]
        .filter(Boolean)
        .join('\n');
      return {
        title: l.Name,
        subtitle,
        arg: `[${l.Name}](${url})`,
        variables: { url, id: l.id, rUrl, tableId: tables.Links.id },
        type: 'file:skipcheck',
        action: { text: `[${l.Name}](${url})` },
        quicklookurl: url,
        text: { largetype, copy: url },
        icon: { path: `media/link${l.Done ? '-done' : ''}.png` },
        match:
          match(l.Name, url) +
          ' ' +
          notes +
          ' ' +
          lists +
          ' ' +
          (l.Tags || []).map((x) => '#' + x.replaceAll(' ', '')).join(' ') +
          ` /${l.Category || ''}`,
        mods: {
          'alt+shift': { subtitle: 'Open record', variables: { url: rUrl } },
          shift: {
            subtitle: 'Send to link copier',
            arg: `[${l.Name}](${url})`,
            variables: { mod: 'save' }
          },
          cmd: done
            ? undefined
            : {
                subtitle: 'Mark as completed 􀃲 ',
                arg: '.',
                icon: { path: 'media/checked.png' },
                variables: {
                  id: l.id,
                  tableId: tables.Links.id,
                  mode: 'complete'
                }
              },
          alt: {
            subtitle: 'Edit record',
            icon: { path: 'media/edit.png' },
            variables: {
              id: l.id,
              title: l.Name,
              url,
              tags: l.Tags?.join(','),
              notes: l.Note,
              category: l.Category,
              table: 'Links',
              done: l.Done ? 1 : 0,
              list:
                l['List-Names']?.length == 1 ? l['List-Names'][0] : undefined,
              mode: 'ask'
            }
          },
          ctrl: {
            subtitle: 'Delete link',
            icon: { path: 'media/delete.png' },
            variables: {
              id: l.id,
              tableId: tables.Links.id,
              mode: 'delete-link'
            }
          },
          fn: {
            subtitle: 'Rebuild cache',
            arg: '__CACHE__',
            icon: { path: 'media/reload.png' },
            variables: { mode: 'cache' }
          }
        }
      };
    });
  items.push({
    title: 'Go Back',
    arg: '__BACK__',
    icon: { path: 'media/back.png' }
  });
  return { items, variables: { prefix: 'Airtable' } };
};

if (import.meta.url === `file://${process.argv[1]}`) {
  switch (process.env.mode) {
    case 'any':
      lookIntoList(process.env.id).then((x) =>
        console.log(JSON.stringify(x, null, 2))
      );
      break;
    case 'all':
    default:
      allLists().then((x) => console.log(JSON.stringify(x, null, 2)));
      break;
  }
}

export { match };
