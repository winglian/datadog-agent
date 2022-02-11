"""
'qa' namespaced tasks
"""

import collections
import os.path

from invoke import task

Card = collections.namedtuple(
    'Card',
    [
        'title',
        'team',
        'description',
        'items',
    ],
)


class QA:
    _snippet_globals = None

    @property
    def snippet_globals(self):
        if self._snippet_globals is None:
            self._snippet_globals = {'cards': [], 'qa': self.qa}
        return self._snippet_globals

    def add_card(self, card):
        self.cards.append(card)

    def qa(self, title, *, team, description=None, items=None):
        self.snippet_globals['cards'].append(
            Card(title=title, team=team, description=description or '', items=items or [])
        )

    def execute_snippet(self, ctx, filename):
        """
        Execute a snippet, given by its filename in the `qa/snippets` directory.
        """
        pathname = os.path.join(ctx.cwd, "qa", "snippets", filename)
        snippet = open(pathname).read()
        snippet = compile(snippet, pathname, 'exec')
        self.cards = []
        exec(snippet, self.snippet_globals)
        for card in self.snippet_globals['cards']:
            print(card)


@task
def check(ctx):
    qa = QA()
    qa.execute_snippet(ctx, "test.py")
