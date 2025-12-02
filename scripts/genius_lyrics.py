#!/usr/bin/env python3
"""
Lyrics fetcher using lyricsgenius library.
Fetches lyrics from Genius.com for a given artist and title.

Usage: genius_lyrics.py <artist> <title>
Output: Lyrics text to stdout, or exit code 1 if not found.

Requires GENIUS_CLIENT_ACCESS_TOKEN environment variable.
"""

import sys
import os


def fetch_lyrics(artist: str, title: str) -> str | None:
    token = os.environ.get("GENIUS_CLIENT_ACCESS_TOKEN")
    if not token:
        print("Error: GENIUS_CLIENT_ACCESS_TOKEN environment variable not set", file=sys.stderr)
        return None

    try:
        from lyricsgenius import Genius
    except ImportError:
        print("Error: lyricsgenius not installed. Run: pip install lyricsgenius", file=sys.stderr)
        return None

    genius = Genius(token, verbose=False, remove_section_headers=True)
    
    # Search for the song
    search_query = f"{title} {artist}"
    songs = genius.search_songs(search_query)
    
    if not songs or not songs.get("hits"):
        return None
    
    # Get the first result
    song = songs["hits"][0]
    url = song["result"]["url"]
    
    # Fetch lyrics from the song URL
    lyrics = genius.lyrics(song_url=url)
    
    if not lyrics:
        return None
    
    # Clean up the lyrics - remove the song title header that lyricsgenius adds
    # It typically adds "SongTitle Lyrics" at the start and "Embed" at the end
    lines = lyrics.split("\n")
    
    # Remove first line if it contains "Lyrics" (the header)
    if lines and "Lyrics" in lines[0]:
        lines = lines[1:]
    
    # Remove trailing embed notice and numbers
    while lines and (lines[-1].strip() == "" or 
                     lines[-1].strip().startswith("Embed") or
                     lines[-1].strip().isdigit() or
                     "You might also like" in lines[-1]):
        lines.pop()
    
    return "\n".join(lines).strip()


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <artist> <title>", file=sys.stderr)
        sys.exit(1)
    
    artist = sys.argv[1]
    title = sys.argv[2]
    
    try:
        lyrics = fetch_lyrics(artist, title)
        if lyrics:
            print(lyrics)
            sys.exit(0)
        else:
            print(f"No lyrics found for '{title}' by '{artist}'", file=sys.stderr)
            sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

