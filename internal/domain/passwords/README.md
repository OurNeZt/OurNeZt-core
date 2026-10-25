# Offline common-password blocklist

`common.txt` contains 1330 unique, lowercase entries. Source: [SecLists 100k-most-used-passwords-NCSC.txt](https://github.com/danielmiessler/SecLists/blob/1a7bb9127eca9e6ff2fc0301c597fe6e16a0cb56/Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt), pinned to commit `1a7bb9127eca9e6ff2fc0301c597fe6e16a0cb56`. The upstream MIT license is included in `LICENSE-SecLists.txt`.

To regenerate the source-derived entries, select the first 1,000 lines plus every entry with at least 15 Unicode code points from that file. Trim surrounding whitespace, lowercase, remove empty entries, deduplicate, and sort. Include these additional predictable/public examples before deduplicating:

- `correct horse battery staple`
- `correcthorsebatterystaple`
- `passwordpassword`
- `passwordpasswordpassword`
- `ourneztpassword123`

The source selection retains common short entries even though they already fail the minimum length. Whitespace-containing entries also fail the whitespace check before blocklist screening. Longer entries cover passwords that pass those checks. Screening compares the whole candidate after lowercasing, never substrings. Hashing preserves the exact original password. The list is embedded at build time and requires no runtime files or network requests.

This is a compact, static blocklist, not comprehensive or live breached-password screening. Review the pinned source and refresh this file deliberately when updating the policy. Keep entries as one lowercase password per line, without comments.
