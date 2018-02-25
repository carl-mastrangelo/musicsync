# Music Syncer

This program synchronizes a music folder to another, converting the files in the 
process.  This program is useful if you have a single directory will all your 
music.   The music may be in different formats and of varying quality, so it may 
not be useful for distribution.

The master copy is assumed to be the one true copy, and the redundant copies track
it.   In my case, my car can only play MP3 files.   Thus, I add music I like to 
the main music directory, and then convert/copy the files over to my car. 

Features:

* Incremental updates
* Parallel conversion
* Dry runs

FFmpeg is needed for this program to work.  By default, libmp3lame is used, with V2
as the default quality setting.  All files, including source MP3s are converted 
as filesize may be a concern.
