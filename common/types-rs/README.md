# Input Types for circuits

A series of separated crates for the input types accepted by circuits as input.

This crate help decoupling circuits with other crates and keep their dependencies neat and controllable. Avoiding to involve crates which is not compatible with the tootlchain of openvm from indirect dependency.

### Code structure
```
types-rs
│
├── base
│
├── circuit
│
├── aggregation
│
<following are layer-oriented crates>
│
├── chunk
│
├── batch
│
└── bundle
```
