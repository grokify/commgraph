# Identity Resolution

Identity resolution merges multiple email addresses belonging to the same person into a single actor entity.

## Why Identity Resolution Matters

Email users often have multiple addresses:

- `john.doe@company.com`
- `jdoe@company.com`
- `john_doe@company.com`
- `johndoe@gmail.com` (personal)

Without resolution, these appear as separate actors, fragmenting the communication graph and producing misleading analysis results.

## How It Works

CommGraph's identity resolution works in layers:

1. **Exact match**: Same email address
2. **Domain normalization**: Handles subdomains and variations
3. **Name matching**: Fuzzy matching of display names
4. **External data**: Pre-defined alias mappings (e.g., Enron employee data)

## Commands

### List Actors

View resolved actors:

```bash
# List all actors
commgraph identity list

# List only internal actors
commgraph identity list --internal

# List only external actors
commgraph identity list --external

# Limit results
commgraph identity list --limit=50

# Output as JSON
commgraph identity list --format=json
```

### View Aliases

See all email addresses associated with an actor:

```bash
commgraph identity aliases jeff.skilling
```

Example output:

```
Actor: jeff.skilling
Display Name: Jeff Skilling
Primary Email: jeff.skilling@enron.com
Internal: true
Title: CEO

Aliases (5):
  jeff.skilling@enron.com (primary)
  jskilli@enron.com
  skilling@enron.com
  jeff_skilling@enron.com
  jeffrey.skilling@enron.com
```

### Resolution Statistics

View identity resolution statistics:

```bash
commgraph identity stats
```

Output:

```
Identity Resolution Statistics:
  Total actors:           5,352
  Internal actors:        148
  External actors:        5,204

  Actors with aliases:    89
  Total aliases:          234
  Average aliases/actor:  2.6

  Resolution sources:
    Exact match:          4,891
    Name matching:        227
    External data:        234
```

## Configuration

### Auto-Create Actors

By default, CommGraph creates actor entries for unknown email addresses:

```yaml
# .commgraph.yaml
identity:
  auto_create: true
```

Set `auto_create: false` to only track actors that match known identities.

### Internal Domains

Specify which domains are internal to your organization:

```yaml
# .commgraph.yaml
source:
  internal_domains:
    - company.com
    - company.org
    - company.internal
```

Or via command line:

```bash
commgraph ingest --internal-domains=company.com,company.org
```

### Enron Employee Data

For Enron corpus analysis, load pre-curated identity data:

```yaml
# .commgraph.yaml
identity:
  load_enron: true
```

Or via command line:

```bash
commgraph pipeline --enron
```

This loads data from the [enron-people](https://github.com/enrondata/enron-people) package, which includes:

- Known aliases for key Enron employees
- Job titles and departments
- Organizational relationships

## Custom Identity Mapping

For custom identity resolution, create a YAML mapping file:

```yaml
# identities.yaml
actors:
  - id: john.doe
    display_name: John Doe
    primary_email: john.doe@company.com
    title: Senior Engineer
    department: Engineering
    aliases:
      - jdoe@company.com
      - john_doe@company.com
      - johndoe@gmail.com
```

Load during ingestion:

```bash
commgraph ingest --identities=identities.yaml --source=/path/to/emails
```

## Best Practices

1. **Define internal domains first**: This ensures proper internal/external classification before analysis.

2. **Review auto-created actors**: After initial ingestion, review the actor list for obvious duplicates.

3. **Use external data when available**: Pre-curated identity data (like Enron) significantly improves analysis accuracy.

4. **Check high-centrality actors**: Actors with unusually high centrality may be unresolved aliases that should be merged.

5. **Iterate**: Identity resolution is often iterative. Analyze, identify issues, add mappings, re-analyze.

## Troubleshooting

### Fragmented Actors

If an actor appears multiple times in results:

1. Check aliases: `commgraph identity aliases <actor-id>`
2. Look for variations in email addresses
3. Add missing aliases to your identity mapping

### Missing Internal Classification

If internal employees appear as external:

1. Verify internal domains are correctly specified
2. Check for domain variations (subdomains, etc.)
3. Add all domain variations to the configuration

### Incorrect Merges

If separate people are incorrectly merged:

1. Review the alias list for the merged actor
2. Remove incorrect aliases from your identity mapping
3. Re-run ingestion with updated mappings
