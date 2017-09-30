# cpbot

Competitive programming contests reminder bot, powered by [https://clist.by](https://clist.by).

Currently only Line bots exist. More to come. Meanwhile, add me on Line!

![](http://qr-official.line.me/M/WU3C0ZeAh9.png)

## Developing

**Prerequisite:**
- Redis
- Line channel with `REPLY_MESSAGE` and `PUSH_MESSAGE` capability. Register here: [https://developers.line.me/en/](https://developers.line.me/en/)
- CList API Key. Get one here: [https://clist.by/api/v1/doc/](https://clist.by/api/v1/doc/)

**Envvars:**
- `CLIST_APIKEY=username:...` without `ApiKey`
- `REDIS_ENDPOINT=host:port`
- `LINE_CHANNEL_SECRET`
- `LINE_CHANNEL_TOKEN`
- `LINE_GREETING_MESSAGE` message to be shown upon join/add as friend event
- `LINE_DAILY_DEFAULT` default schedule for daily reminder
- `LINE_DAILY_PERIOD` period of cron job of sending daily reminder. Suggested: 1800 (half an hour)
- `LINE_MAX_MESSAGE_LENGTH` max length of a message. Limit from Line is 2000. Suggested: 1000.

**Running locally:**
Use realize to develop locally and watch for file changes.
```bash
go get github.com/tockins/realize

CLIST_API_KEY=... \
MORE_ENVVAR=... \
realize run
```

## Deploying

Line requires SSL for all their webhooks. I suggest deploying to [Heroku](https://heroku.com).
After that, set your line webhook to `https://url/line/callback`
