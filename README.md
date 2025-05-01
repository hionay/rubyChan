# rubyChan

A simple Matrix bot written in Go for personal use. 

We use it in our friends' space(NerdHorizon) to share quotes, search Google, and for some other fun stuff.

In time, I'll add more features, but for now, it's just a simple bot.

## Features

- `!g <query>` — Google search and return top result  
- `!weather <location>` — Current weather via WeatherAPI.com  
- `!joke` — Random joke via JokeAPI.dev
- `!calc <expr>` — Evaluate a math expression  
- `!roulette` — Russian roulette (1 in 6 chance)  
- `!remindme in <duration> <message>` — In-chat reminder  
- `!remindme list` - List pending reminders
- `!remindme cancel <id>` - Cancel a reminder
- `!quote <N> [comment]` — Quote the last N messages and post to our quotes API  
- `!help` — Show available commands  