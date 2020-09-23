# GitOps-ish
> follow me on github to learn how I'm _really_ feeling...

## Trending Buzzwords
Everything is fun an entertaining when it has cool words associated with it

> How does one find happiness in the abyss?

## Checking if everything is ok?

```bash
curl localhost:9999/are-you-ok
ok

curl  localhost:9999/are-you-ok\?really=true
permission denied

curl --Header Authorization: <friendship-token> localhost:6391/are-you-ok?really=true
{
410
msg: "wake me up when september ends"
}
```