This describes a new music organisation feature, "keywords".

Every track has "Keywords", like "Instrumental", "Hiphop", "Running". These are strings. They're canonically stored on the actual music file (although might be cached in the database). I think we can use the existing Tag functionality for this?

A track's keywords are displayed in the song listing table wherever it appears (album view, playlist view, song list, etc) via the "Columns To Display" dropdown. There might be a few so we use a small font.

An Album doesn't have keywords directly, but it's considered to have the union of all of its tracks' keywords. Playlists work the same way. These are shown in the header (under the star rating).

Clicking on the keywords shown for a track in the track list opens a modal edit window showing existing keywords (click to remove), all other keywords that are used anywhere (click to add), and a free-text box (type and press enter to add a new keyword, with autocomplete against existing keywords).

Clicking on the keywords shown in the album/playlist/etc header opens the same modal, but clicking 'save' applies the new keyword list to every track in the album/playlist/etc, overwriting the old value.


