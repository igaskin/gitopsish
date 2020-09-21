# GitOps-ish
> follow me on github to learn how I'm _really_ feeling...

## Trending Buzzwords
Everything is fun an entertaining when it has cool words associated with it

> How does oen find happiness in the abyss?

## Checking if everything is ok?

```
curl localhost:6391/are-you-ok

200

curl localhost:6391/are-you-ok?really=true

403


curl --Header Authorization: <friendship-token> localhost:6391/are-you-ok?really=true
{
410
msg: "wake me up when september ends"
}
```



