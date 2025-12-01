#!/usr/bin/env python3
"""
Audio energy analyzer using Essentia library.
Outputs JSON with BPM, beat loudness, average loudness, and danceability.
Usage: essentia_analyze.py <audio_file> <output_json>
"""

import sys
import json

def analyze(input_file, output_file):
    import essentia.standard as es
    
    # Load audio file
    loader = es.MonoLoader(filename=input_file)
    audio = loader()
    
    # Get sample rate for rhythm analysis
    sample_rate = 44100  # MonoLoader default
    
    # Rhythm analysis (BPM and beats)
    rhythm_extractor = es.RhythmExtractor2013(method="multifeature")
    bpm, beats, beats_confidence, _, beats_intervals = rhythm_extractor(audio)
    
    # Beat loudness
    beat_loudness_values = []
    if len(beats) > 1:
        for i in range(len(beats) - 1):
            start_sample = int(beats[i] * sample_rate)
            end_sample = int(beats[i + 1] * sample_rate)
            if end_sample <= len(audio):
                segment = audio[start_sample:end_sample]
                if len(segment) > 0:
                    rms = es.RMS()(segment)
                    beat_loudness_values.append(rms)
    
    beats_loudness_mean = sum(beat_loudness_values) / len(beat_loudness_values) if beat_loudness_values else 0.0
    
    # Average loudness
    average_loudness = es.Loudness()(audio)
    # Normalize to 0-1 range (loudness is typically negative dB, we'll use a sigmoid-like transform)
    # Typical range is -60 to 0 dB, map to 0-1
    normalized_loudness = max(0, min(1, (average_loudness + 60) / 60))
    
    # Danceability
    danceability, _ = es.Danceability()(audio)
    
    # Build output structure matching what the Go code expects
    result = {
        "rhythm": {
            "bpm": float(bpm),
            "beats_loudness": {
                "mean": float(beats_loudness_mean)
            }
        },
        "lowlevel": {
            "average_loudness": float(normalized_loudness)
        },
        "highlevel": {
            "danceability": {
                "all": {
                    "danceable": float(danceability)
                }
            }
        }
    }
    
    with open(output_file, 'w') as f:
        json.dump(result, f, indent=2)
    
    return 0

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <input_audio> <output_json>", file=sys.stderr)
        sys.exit(1)
    
    try:
        sys.exit(analyze(sys.argv[1], sys.argv[2]))
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

