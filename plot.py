#!/usr/bin/env python3

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import timedelta, datetime

# this script loads data from a csv file
# with headers id, created_at, time, title, url, author, ndescendants, score, rank
# and then saves a plot with score, ndescendants and rank for each id

# load data from csv file
df = pd.read_csv('hacker_news.csv', index_col='created_at')

# group pandas dataframe by id
grouped = df.groupby(['id'])

# create one chart per id and plot score, ndescendants and rank in each chart
for [hn_id], group in grouped:
    # sort group by created_at ascending
    group = group.sort_values(by='created_at', ascending=True)

    # this is the time when the item was created on HN
    item_created_at = datetime.utcfromtimestamp(group['time'].values[0])

    # use relative time for x axis
    def date_to_relative(d1):
        date_fmt = '%Y-%m-%d %H:%M:%S'
        current = datetime.strptime(d1, date_fmt)
        return (current - item_created_at) / timedelta(hours=1)

    group.index = group.index.map(date_to_relative)

    # title generation
    hn_item_title = group['title'].values[0]
    hn_item_url = group['url'].values[0]
    hn_item_link = f'https://news.ycombinator.com/item?id={hn_id}'
    plot_title = f'{hn_item_title}\n{hn_item_url}\n{hn_item_link}'

    fig, ax1 = plt.subplots(figsize=(10, 5))

    ax1.set_title(plot_title)
    ax1.set_xlabel('hours')
    ax1.set_ylabel('score, comments')
    ax1.plot(group['score'], label='score', color='blue')
    ax1.plot(group['ndescendants'], label='comments', color='orange')
    ax1.legend()

    # show every 50th date
    # TODO: do something more clever here
    plt.xticks(group.index[::50], rotation=45)

    ax2 = ax1.twinx()
    ax2.set_ylabel('rank')
    ax2.set_ylim(1, 30)
    ax2.plot(group['rank'], label='rank', color='green')
    ax2.legend(loc='upper right')

    plt.tight_layout()
    plt.savefig(f'hn_{hn_id}.png')
