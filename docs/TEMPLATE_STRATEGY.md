# Template Collection Strategy for v1.0.0

This document outlines the comprehensive template library we'll build before tagging v1.0.0.

## Goals

1. **Coverage** - Address major HPC/scientific computing domains
2. **Quality** - Production-ready, well-tested templates
3. **Variation** - Provide options for different scales and use cases
4. **Documentation** - Clear descriptions and use case guidance

## Current Status (Baseline)

### Existing Templates (5 total)
- ✅ seeds/library/bioinformatics.yaml
- ✅ seeds/library/machine-learning.yaml
- ✅ seeds/library/computational-chemistry.yaml
- ✅ seeds/examples/minimal.yaml
- ✅ seeds/examples/starter.yaml

##Target: 25-30 Templates Across 8 Categories

---

## Category 1: Bioinformatics (4-5 templates)

### ✅ bioinformatics.yaml (EXISTS)
General genomics with samtools, bwa, gatk, blast+

### NEW: rna-seq.yaml
**Use Case:** RNA sequencing analysis
**Software:**
- Alignment: STAR, hisat2, bowtie2
- Quantification: salmon, kallisto, featureCounts
- Differential expression: DESeq2, edgeR (via R)
- QC: fastqc, multiqc, RSeQC
- Languages: python@3.10, r@4.2.0
**Instances:** Memory-optimized (r5.4xlarge, r5.8xlarge)

### NEW: single-cell.yaml
**Use Case:** Single-cell RNA-seq analysis (10x Genomics, etc.)
**Software:**
- cellranger
- seurat (via R)
- scanpy (via Python)
- velocyto
- python@3.10, r@4.2.0
**Instances:** High-memory (r5.12xlarge, r5.24xlarge)

### NEW: metagenomics.yaml
**Use Case:** Microbiome and metagenomic analysis
**Software:**
- Assembly: megahit, spades
- Annotation: prokka, prodigal
- Taxonomic classification: kraken2, metaphlan
- Functional profiling: humann3
- Quality: fastp, fastqc
**Instances:** Compute + memory mix

### NEW: structural-biology.yaml
**Use Case:** Protein structure prediction and molecular dynamics
**Software:**
- alphafold
- rosetta
- pymol
- chimera
- blast-plus
**Instances:** GPU + CPU (g4dn for alphafold)

---

## Category 2: Machine Learning / AI (5-6 templates)

### ✅ machine-learning.yaml (EXISTS)
General ML with PyTorch, TensorFlow, CUDA

### NEW: llm-training.yaml
**Use Case:** Large language model training
**Software:**
- transformers
- accelerate
- deepspeed
- flash-attention
- wandb
- python@3.10, cuda@11.8
**Instances:** Multi-GPU (p3.16xlarge, p4d.24xlarge)

### NEW: computer-vision.yaml
**Use Case:** Image classification, object detection
**Software:**
- pytorch-vision
- detectron2
- mmdetection
- opencv
- pillow, albumentations
**Instances:** GPU (g4dn.xlarge-4xlarge, p3.2xlarge)

### NEW: reinforcement-learning.yaml
**Use Case:** RL training for robotics, games
**Software:**
- ray[rllib]
- stable-baselines3
- gymnasium
- mujoco
- pytorch
**Instances:** CPU-heavy (c5.18xlarge)

### NEW: ml-inference.yaml
**Use Case:** Model serving and batch inference
**Software:**
- torchserve
- triton-inference-server
- onnx-runtime
- tensorrt
**Instances:** Mixed GPU/CPU (g4dn.xlarge, c5.2xlarge)

### NEW: automl.yaml
**Use Case:** Automated machine learning and hyperparameter tuning
**Software:**
- optuna
- ray[tune]
- auto-sklearn
- keras-tuner
**Instances:** Many small instances (c5.xlarge × 50)

---

## Category 3: Computational Chemistry / Physics (4 templates)

### ✅ computational-chemistry.yaml (EXISTS)
General chemistry with GROMACS, LAMMPS, Quantum ESPRESSO

### NEW: molecular-dynamics.yaml
**Use Case:** Advanced MD simulations
**Software:**
- gromacs@2023.1 +cuda
- amber
- namd@3.0
- vmd
**Instances:** GPU-optimized (g4dn.12xlarge, p3.8xlarge)

### NEW: quantum-chemistry.yaml
**Use Case:** Electronic structure calculations
**Software:**
- nwchem
- psi4
- orca
- gaussian (if licensed)
- molpro
**Instances:** High-memory (r5.24xlarge, x1e.32xlarge)

### NEW: materials-science.yaml
**Use Case:** DFT, materials modeling
**Software:**
- vasp (if licensed)
- quantum-espresso
- siesta
- lammps
- pymatgen
**Instances:** Compute-optimized (c5.24xlarge)

---

## Category 4: Data Science / Analytics (4 templates)

### NEW: data-science.yaml
**Use Case:** General data analysis, statistics
**Software:**
- python@3.10 with pandas, numpy, scipy
- r@4.2.0 with tidyverse
- jupyter-lab
- dask for parallel computing
- plotly, seaborn for viz
**Instances:** General purpose (m5.4xlarge)

### NEW: big-data-spark.yaml
**Use Case:** Spark-based big data processing
**Software:**
- spark@3.4.0
- hadoop@3.3.0
- python@3.10 (pyspark)
- scala@2.12
**Instances:** Memory-optimized (r5.12xlarge)

### NEW: time-series.yaml
**Use Case:** Financial modeling, forecasting
**Software:**
- prophet
- statsmodels
- scikit-learn
- tensorflow (for LSTM)
- python@3.10, r@4.2.0
**Instances:** Compute-optimized (c5.9xlarge)

### NEW: geospatial.yaml
**Use Case:** GIS analysis, satellite imagery
**Software:**
- gdal
- geopandas
- rasterio
- qgis
- grass
**Instances:** Memory-optimized (r5.8xlarge)

---

## Category 5: Engineering / CFD (3 templates)

### NEW: cfd-openfoam.yaml
**Use Case:** Computational fluid dynamics
**Software:**
- openfoam@2306
- paraview
- openmpi@4.1.4
- scotch, metis (mesh partitioning)
**Instances:** Compute-optimized (c5.18xlarge, c5n.18xlarge)

### NEW: fem-analysis.yaml
**Use Case:** Finite element analysis
**Software:**
- fenics
- elmer
- gmsh
- paraview
**Instances:** Compute + memory (m5.12xlarge)

### NEW: numerical-simulation.yaml
**Use Case:** General engineering simulations
**Software:**
- octave (MATLAB alternative)
- scilab
- python@3.10 with scipy
- gnuplot
**Instances:** General purpose (m5.8xlarge)

---

## Category 6: General Purpose / Starter (4 templates)

### ✅ minimal.yaml (EXISTS)
Bare minimum cluster

### ✅ starter.yaml (EXISTS)
Basic cluster with common tools

### NEW: development.yaml
**Use Case:** Software development and testing
**Software:**
- gcc, clang, llvm
- cmake, make, ninja
- git, svn
- python, perl, ruby
- vim, emacs
**Instances:** Mixed (t3.xlarge head, c5.2xlarge compute)

### NEW: jupyter-hub.yaml
**Use Case:** Multi-user Jupyter environment
**Software:**
- jupyterhub
- jupyter-lab
- python@3.10 with data science stack
- r-kernel
- julia
**Instances:** General purpose (m5.4xlarge)

---

## Category 7: Specialized Scientific (3 templates)

### NEW: climate-modeling.yaml
**Use Case:** Climate and weather simulations
**Software:**
- wrf (Weather Research and Forecasting)
- cesm (Community Earth System Model)
- netcdf, hdf5
- ncview, panoply
**Instances:** Compute-heavy (c5.24xlarge)

### NEW: astronomy.yaml
**Use Case:** Astronomical data analysis
**Software:**
- astropy
- ds9
- sextractor
- photutils
- casa (radio astronomy)
**Instances:** Memory + compute (r5.12xlarge)

### NEW: systems-biology.yaml
**Use Case:** Pathway analysis, networks
**Software:**
- cytoscape
- celldesigner
- copasi
- r@4.2.0 with Bioconductor
**Instances:** General purpose (m5.8xlarge)

---

## Category 8: Cost-Optimized / Educational (2-3 templates)

### NEW: spot-compute.yaml
**Use Case:** Budget-conscious batch workloads
**Features:**
- 100% spot instances
- Checkpointing recommended
- Low-cost instance types (c5.large, c5.xlarge)
**Software:** Minimal set based on use case

### NEW: educational.yaml
**Use Case:** Teaching, workshops, training
**Features:**
- Small scale (max 5 nodes)
- Mix of tools from different domains
- Documentation-focused
**Instances:** t3.medium, t3.large (burstable)

### NEW: benchmarking.yaml
**Use Case:** Performance testing, comparisons
**Software:**
- hpl (LINPACK)
- hpcg
- stream
- ior, mdtest (I/O benchmarks)
- iperf3 (network)
**Instances:** Variety for testing

---

## Implementation Priority

### Phase 1: Core Domains (Week 1)
High-impact templates users need immediately:
1. rna-seq.yaml
2. llm-training.yaml
3. molecular-dynamics.yaml
4. data-science.yaml
5. cfd-openfoam.yaml

### Phase 2: Specialized (Week 2)
Domain-specific needs:
6. single-cell.yaml
7. computer-vision.yaml
8. quantum-chemistry.yaml
9. big-data-spark.yaml
10. jupyter-hub.yaml

### Phase 3: Extended Coverage (Week 3)
11. metagenomics.yaml
12. reinforcement-learning.yaml
13. materials-science.yaml
14. time-series.yaml
15. fem-analysis.yaml

### Phase 4: Polish & Documentation (Week 4)
16-25. Remaining templates
- Template testing and validation
- Documentation for each template
- README for template library

---

## Template Structure Standards

Each template should include:

1. **Header Comment Block**
   ```yaml
   # [Template Name]
   # Use Case: [One-line description]
   # Domain: [Category]
   # Optimized for: [Specific workloads]
   # Included software: [Key packages]
   # Instance recommendations: [Instance types and why]
   # Estimated costs: [Rough hourly cost for typical usage]
   ```

2. **Clear Section Organization**
   - cluster: name, region
   - compute: head_node, queues (with comments explaining each)
   - software: spack_packages (organized by category)
   - users: example users
   - data: s3_mounts (with descriptive names)

3. **Inline Comments**
   - Explain queue choices
   - Note software dependencies
   - Provide usage tips

4. **Companion README**
   - Getting started guide
   - Example workflows
   - Common commands
   - Troubleshooting

---

## Success Metrics

- ✅ 25-30 templates covering 8 major categories
- ✅ Each template tested with `pctl create` (dry run minimum)
- ✅ Documentation for each template
- ✅ Template library README with decision tree
- ✅ Cross-reference with persona use cases

---

## Post-v1.0.0 Template Enhancements

Ideas for future template features:
- Template variants (small/medium/large scale)
- Industry-specific templates (pharma, finance, energy)
- Container-based templates
- PCS-compatible templates (v2.0)
- Community-contributed templates

---

## Notes

- Focus on **AWS Spack buildcache compatibility** - prefer packages with binary builds
- Include **cost estimates** in template comments
- Provide **scaling guidance** (when to use spot, when to use on-demand)
- Link to **persona walkthroughs** where applicable
- Test templates with AMI building to ensure software installs correctly
