# GPC Domain Language

Project-specific terms used inside `gpc` (GORM Preload Checker). These are the names the code, tests, and ADRs should use consistently.

## Language

**Relation**:
A GORM association declared as a field on a model struct (e.g. `User.Profile`). The string passed to `db.Preload("...")` names one or more relations joined by dots.
_Avoid_: association (GORM's external term), edge, link

**Relation Path**:
A dotted string identifying a chain of relations to traverse from a model (e.g. `"User.Profile.Address"`). Each segment must resolve to a struct field on the previous segment's struct type.
_Avoid_: preload string, dotted name, path expression

**Model**:
The Go struct type a Preload chain ultimately loads into — the type unwrapped from the argument to `Find` / `First` / `FirstOrCreate`. Always a named struct (after pointer/slice peeling).
_Avoid_: entity, record, target type

**Chain**:
A single `db.Preload(...).Preload(...).Find(&x)` expression treated as one analysis unit. Carries every Preload call plus the terminal Find-family call that pins the **Model**.
_Avoid_: pipeline, query, expression

**Verification** (formerly "validation"):
Deciding whether every **Relation Path** in a **Chain** resolves against the **Chain**'s **Model** via Go type information. Lives in the `internal/relations` package.
_Avoid_: validation (overloaded with input validation), checking

## Relationships

- A **Chain** has one **Model** and one or more **Relation Paths**.
- A **Relation Path** is a sequence of **Relations** evaluated left-to-right against the **Model**.
- **Verification** consumes a **Chain** and produces a result per **Relation Path**.

## Example dialogue

> **Dev:** "If a **Chain** has three **Preload** calls, do we run **Verification** three times?"
> **Reviewer:** "One **Verification** per **Relation Path** — so yes, three. Each walks the **Model**'s type graph independently. The **Chain** only matters because it pins the **Model**."

## Flagged ambiguities

- "validation" was used for both relation-path verification and unrelated input checks — resolved: this project uses **Verification** for the relation-path concern; "validation" is reserved for general input checks (none currently exist).
