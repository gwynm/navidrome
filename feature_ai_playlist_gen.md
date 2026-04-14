New feature: AI playlist generation

## Basics

We add a new link to the sidebar, "Robot DJ".

It shows a form with Name and Criteria (multiline textbox), option buttons for "strict" and "vibes", and a Preview button.

Preview shows the rules that will be used, and the resulting songs, below the form (just artist / track title, one per line), plus a "Confirm" button that creates the playlist and redirects to it.

The Playlist's description should include the criteria and the rules (filter string returned from the llm).

## Strict mode

We create a Prompt to give to an LLM (Opus4.6, accessed via OpenRouter - we'll need an OpenRouter API key in the env), instructing it the shape of the data we have (including sample values for things like energy and mood), the user's criteria, and a format to return a list of filters in. The LLM won't select songs directly, it'll just tell us how to select them.

We pass that to the LLM, then we parse its response, generating filters that we can apply against our database, producing a list of tracks.

We should write some tests and iterate on this, adjusting the prompt with various test criteria until the LLM almost always returns a set of filters that we can parse and apply correctly.

## Vibes mode

Instead of creating a prompt, we assemble a large, flat, CSV file containing all the relevant information we have in the database. One row per track, but _many_ columns, including all the metadata we have. The first row should be column headings so the LLM knows how to parse it.

Then we give this file, plus the criteria, to the LLM, and instruct it to return a list of track ids that we should put in the playlist. We should tell it to use both the data in the CSV file, and anything it knows by itself.

The "rules" shows in Preview is just "(vibes)" in this case.

This call might be expensive, so we should stop and check with a human about the size of the CSV file after we've generated it but before we send it for the first time.

