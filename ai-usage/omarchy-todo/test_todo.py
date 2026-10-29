import importlib.machinery
import importlib.util
import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

loader = importlib.machinery.SourceFileLoader('todo', str(Path(__file__).parent / 'bin/todo'))
spec = importlib.util.spec_from_loader(loader.name, loader)
todo = importlib.util.module_from_spec(spec)
loader.exec_module(todo)

class TodoTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.env = patch.dict(os.environ, TODO_DB=self.temp.name + '/tasks.db')
        self.env.start()
        self.notify = patch.object(todo, 'notify')
        self.notify.start()

    def tearDown(self):
        self.notify.stop()
        self.env.stop()
        self.temp.cleanup()

    def run_cli(self, *args):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            todo.main(list(args))
        return out.getvalue()

    def status(self):
        return json.loads(self.run_cli('status'))

    def test_lifecycle_and_counts(self):
        self.run_cli('add', 'Today task')
        self.run_cli('add', 'Later task', '--due', 'tomorrow')
        self.run_cli('add', 'Old task', '--due', '2000-01-01')
        self.assertEqual(self.status(), {'total': 3, 'today': 1, 'overdue': 1})
        self.run_cli('priority', '1', 'high')
        self.run_cli('change', '1', '--title', 'Changed title')
        row = json.loads(self.run_cli('list', '--today', '--json'))[0]
        self.assertEqual((row['title'], row['priority']), ('Changed title', 'high'))
        self.run_cli('complete', '1')
        self.assertEqual(self.status()['today'], 0)
        self.run_cli('reopen', '1')
        self.assertEqual(self.status()['today'], 1)
        self.run_cli('change', '1', '--due', 'none')
        self.assertEqual(self.status()['today'], 0)
        self.run_cli('delete', '3')
        self.assertEqual(self.status()['overdue'], 0)
        self.run_cli('add', 'New task')
        self.assertIn('4', self.run_cli('list'))

    def test_status_total_includes_future_and_unscheduled(self):
        self.assertEqual(self.status(), {'total': 0, 'today': 0, 'overdue': 0})
        for name, due in [('Today', 'today'), ('Future', 'tomorrow'), ('Unscheduled', 'none'), ('Overdue', '2000-01-01')]:
            self.run_cli('add', name, '--due', due)
        self.assertEqual(self.status(), {'total': 4, 'today': 1, 'overdue': 1})
        self.run_cli('complete', '1')
        self.run_cli('complete', '4')
        self.assertEqual(self.status(), {'total': 2, 'today': 0, 'overdue': 0})

    def test_day_first_dates(self):
        self.run_cli('add', 'September task', '--due', '08/09/2026')
        rows = json.loads(self.run_cli('list', '--json'))
        self.assertEqual(rows[0]['due'], '2026-09-08')
        self.run_cli('change', '1', '--due', '29/02/2028')
        self.assertEqual(json.loads(self.run_cli('list', '--json'))[0]['due'], '2028-02-29')
        for invalid in ['29/02/2026', '31/09/2026', '09/25/2026']:
            with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
                self.run_cli('change', '1', '--due', invalid)
        self.assertEqual(json.loads(self.run_cli('list', '--json'))[0]['due'], '2028-02-29')

    def test_interactive_crud_loop(self):
        replies = iter(['c', 'Interactive task', '08/09/2026', 'high',
                        '1', 'r', '', 'u', 'open', 'Updated task', 'none', 'low',
                        'u', 'done', 'Shipped it',
                        'd', 'n', 'd', '', ''])
        with patch('builtins.input', side_effect=lambda *args: next(replies)):
            output = self.run_cli('interactive')
        self.assertIn('Added task 1.', output)
        self.assertIn('[B] Back to list', output)
        self.assertIn('Todo #1', output)
        self.assertIn('Updated todo #1.', output)
        self.assertIn('Deletion cancelled.', output)
        self.assertIn('Deleted todo #1: Updated task', output)
        self.assertEqual(json.loads(self.run_cli('list', '--all', '--json')), [])

    def test_interactive_invalid_input_and_defaults(self):
        replies = iter(['x', 'abc', '999', 'c', '', 'Valid',
                        '31/02/2026', '', 'urgent', '', '1', 'z', 'u', 'open', '', '', '', 'b', ''])
        with patch('builtins.input', side_effect=lambda *args: next(replies)):
            output = self.run_cli('interactive')
        task = json.loads(self.run_cli('list', '--json'))[0]
        self.assertEqual((task['title'], task['priority']), ('Valid', 'normal'))
        self.assertEqual(task['due'], todo.date('today'))
        self.assertIn('does not exist', output)

    def test_interactive_cancel_is_atomic(self):
        self.run_cli('add', 'Original')
        with patch('builtins.input', side_effect=['1', 'u', 'open', 'Changed', KeyboardInterrupt(), '']):
            self.run_cli('interactive')
        self.assertEqual(json.loads(self.run_cli('list', '--json'))[0]['title'], 'Original')

    def test_update_done_records_completion_and_outcome(self):
        self.run_cli('add', 'Ship it')
        replies = iter(['1', 'u', 'done', 'Shipped to prod', ''])
        with patch('builtins.input', side_effect=lambda *args: next(replies)):
            self.run_cli('interactive')
        row = json.loads(self.run_cli('list', '--all', '--json'))[0]
        self.assertEqual(row['completed'], 1)
        self.assertEqual(row['outcome'], 'Shipped to prod')
        self.assertTrue(row['completed_at'].startswith(todo.date('today')))
        self.assertIn('Shipped to prod', todo.task_details(row))

    def test_update_done_blank_outcome_and_reopen_clears(self):
        self.run_cli('add', 'Quick task')
        replies = iter(['1', 'u', 'done', '', ''])
        with patch('builtins.input', side_effect=lambda *args: next(replies)):
            self.run_cli('interactive')
        row = json.loads(self.run_cli('list', '--all', '--json'))[0]
        self.assertEqual(row['completed'], 1)
        self.assertIsNone(row['outcome'])
        self.assertTrue(row['completed_at'].startswith(todo.date('today')))
        replies = iter(['1', 'u', 'open', 'Quick task v2', '', '', ''])
        with patch('builtins.input', side_effect=lambda *args: next(replies)):
            self.run_cli('interactive')
        row = json.loads(self.run_cli('list', '--json'))[0]
        self.assertEqual((row['completed'], row['title']), (0, 'Quick task v2'))
        self.assertIsNone(row['completed_at'])
        self.assertIsNone(row['outcome'])

    def test_complete_and_reopen_maintain_completion_fields(self):
        self.run_cli('add', 'Task one')
        self.run_cli('complete', '1', '--outcome', 'Done via CLI')
        row = json.loads(self.run_cli('list', '--all', '--json'))[0]
        self.assertEqual((row['completed'], row['outcome']), (1, 'Done via CLI'))
        self.assertTrue(row['completed_at'].startswith(todo.date('today')))
        self.run_cli('reopen', '1')
        row = json.loads(self.run_cli('list', '--json'))[0]
        self.assertEqual(row['completed'], 0)
        self.assertIsNone(row['completed_at'])
        self.assertIsNone(row['outcome'])

    def test_legacy_database_gains_new_columns(self):
        import sqlite3
        db = sqlite3.connect(os.environ['TODO_DB'])
        db.execute('CREATE TABLE tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, due TEXT, priority TEXT NOT NULL, completed INTEGER NOT NULL DEFAULT 0)')
        db.execute("INSERT INTO tasks(title,due,priority,completed) VALUES ('Old','2026-01-01','normal',0)")
        db.commit()
        db.close()
        rows = json.loads(self.run_cli('list', '--json'))
        self.assertEqual(rows[0]['title'], 'Old')
        self.assertIsNone(rows[0]['completed_at'])
        self.assertIsNone(rows[0]['outcome'])

    def test_version_and_author_are_visible_and_in_sync(self):
        manifest = json.loads((Path(__file__).parent / 'manifest.json').read_text())
        self.assertEqual(todo.VERSION, manifest['version'])
        self.assertEqual(todo.AUTHOR, manifest['author'])
        widget = (Path(__file__).parent / 'Widget.qml').read_text()
        self.assertIn('v' + todo.VERSION, widget)
        self.assertIn(todo.AUTHOR, widget)
        with patch('builtins.input', side_effect=['']):
            output = self.run_cli('interactive')
        self.assertIn('v' + todo.VERSION, output)
        self.assertIn(todo.AUTHOR, output)

    def test_read_cursor_restored_on_interrupt(self):
        out = io.StringIO()
        with patch.object(out, 'isatty', return_value=True), patch.object(todo.sys.stdin, 'isatty', return_value=True), patch.dict(os.environ, TERM='xterm'), contextlib.redirect_stdout(out), patch.object(todo.time, 'sleep', side_effect=KeyboardInterrupt()):
            with self.assertRaises(KeyboardInterrupt):
                todo.read_task({'id': 1, 'title': 'Read me', 'due': None, 'priority': 'normal', 'completed': 0})
        self.assertIn('\033[1 q', out.getvalue())
        self.assertTrue(out.getvalue().endswith('\033[0 q'))

    def test_invalid_input_does_not_mutate(self):
        self.run_cli('add', 'Keep me')
        for args in [('complete', '999'), ('change', '1'), ('add', ' '), ('add', 'Bad date', '--due', '2026-02-30')]:
            with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
                self.run_cli(*args)
        self.assertEqual(len(json.loads(self.run_cli('list', '--json'))), 1)

if __name__ == '__main__':
    unittest.main()
