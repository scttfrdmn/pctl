# Feature Request: AOCC and Intel Compiler Support

## Status: Planned

## Priority: High

## Overview

Add support for AMD Optimizing C/C++ Compiler (AOCC) and Intel oneAPI compilers in petal seeds. These commercial/optimized compilers are critical for HPC workloads and can provide 10-30% performance improvements over GCC on specific architectures.

## Background

### Current State
petal supports GCC via Spack:
```yaml
software:
  spack_packages:
    - gcc@11.3.0
    - gcc@12.2.0
```

### User Need
HPC users need optimized compilers for:
- **AOCC**: Optimized for AMD EPYC processors (common in AWS - c5a, m5a, r5a instances)
- **Intel oneAPI**: Optimized for Intel processors (most AWS instances), replaces Intel Parallel Studio
- **Performance**: 10-30% speedup for compute-intensive workloads
- **Vendor libraries**: AMD AOCL and Intel MKL integrated with respective compilers

## Supported Compilers

### 1. AMD Optimizing C/C++ Compiler (AOCC)

**Spack Package**: `aocc@4.0.0` or later

**Features**:
- Optimized for AMD EPYC (Zen 2, Zen 3, Zen 4 architectures)
- Clang/LLVM-based with AMD-specific optimizations
- Includes Fortran compiler (flang)
- Free for use (requires EULA acceptance)

**AWS Instance Types**:
- c5a, c6a, c7a (compute-optimized AMD)
- m5a, m6a, m7a (general-purpose AMD)
- r5a, r6a, r7a (memory-optimized AMD)
- hpc7a (HPC-optimized AMD EPYC)

**Typical Performance Gains**:
- Molecular dynamics: 15-25%
- CFD simulations: 10-20%
- Bioinformatics: 10-15%

### 2. Intel oneAPI Compilers

**Spack Packages**:
- `intel-oneapi-compilers@2024.0.0` (C/C++/Fortran)
- `intel-oneapi-mpi@2021.11.0` (MPI implementation)
- `intel-oneapi-mkl@2024.0.0` (Math Kernel Library)

**Features**:
- Optimized for Intel processors (Xeon, Core)
- Includes classic compilers (icc, icpc, ifort) and next-gen (icx, icpx, ifx)
- Integrated with Intel MKL, IPP, TBB
- Free for use (requires registration)

**AWS Instance Types**:
- c5, c6i, c7i (compute-optimized Intel)
- m5, m6i, m7i (general-purpose Intel)
- r5, r6i, r7i (memory-optimized Intel)
- hpc6id, hpc7i (HPC-optimized Intel)

**Typical Performance Gains**:
- Linear algebra (MKL): 20-40%
- Quantum chemistry: 15-25%
- Engineering simulations: 10-20%

## Implementation

### 1. Spack Package Support

Both compilers are available in Spack:

```bash
# Check availability
spack info aocc
spack info intel-oneapi-compilers

# Install from Spack
spack install aocc@4.0.0
spack install intel-oneapi-compilers@2024.0.0
```

### 2. Example Seed: AOCC Stack

**File**: `seeds/library/compute-amd-aocc.yaml`

```yaml
cluster:
  name: amd-compute
  region: us-west-2

compute:
  head_node: c6a.2xlarge  # AMD EPYC
  queues:
    - name: compute
      instance_types:
        - c6a.8xlarge   # 32 vCPU AMD EPYC
        - c6a.16xlarge  # 64 vCPU AMD EPYC
      min_count: 0
      max_count: 10

software:
  spack_packages:
    # AMD Optimizing Compiler
    - aocc@4.0.0

    # MPI built with AOCC
    - openmpi@4.1.4 %aocc@4.0.0

    # AMD Optimized Libraries
    - aocl-blis@4.0      # Linear algebra
    - aocl-libflame@4.0  # LAPACK
    - aocl-fftw@4.0      # FFT

    # Scientific software built with AOCC
    - lammps@20230802 %aocc@4.0.0
    - gromacs@2023.1 %aocc@4.0.0

    # Python with optimized NumPy
    - python@3.10
    - py-numpy@1.24.0 ^openblas %aocc@4.0.0
```

### 3. Example Seed: Intel oneAPI Stack

**File**: `seeds/library/compute-intel-oneapi.yaml`

```yaml
cluster:
  name: intel-compute
  region: us-east-1

compute:
  head_node: c6i.2xlarge  # Intel Xeon
  queues:
    - name: compute
      instance_types:
        - c6i.8xlarge   # 32 vCPU Intel Xeon
        - c6i.16xlarge  # 64 vCPU Intel Xeon
      min_count: 0
      max_count: 10

software:
  spack_packages:
    # Intel oneAPI Compiler Suite
    - intel-oneapi-compilers@2024.0.0

    # Intel MPI
    - intel-oneapi-mpi@2021.11.0

    # Intel Math Kernel Library
    - intel-oneapi-mkl@2024.0.0

    # Scientific software built with Intel
    - lammps@20230802 %oneapi@2024.0.0
    - quantum-espresso@7.2 %oneapi@2024.0.0 ^intel-oneapi-mkl

    # Python with Intel-optimized NumPy
    - python@3.10
    - py-numpy@1.24.0 ^intel-oneapi-mkl
```

### 4. Example Seed: Multi-Compiler Environment

**File**: `seeds/library/compute-multi-compiler.yaml`

```yaml
cluster:
  name: multi-compiler
  region: us-west-2

compute:
  head_node: c6a.2xlarge
  queues:
    # AMD queue with AOCC
    - name: amd
      instance_types: [c6a.8xlarge]
      min_count: 0
      max_count: 5

    # Intel queue (if mixed instance types supported)
    - name: intel
      instance_types: [c6i.8xlarge]
      min_count: 0
      max_count: 5

software:
  spack_packages:
    # Multiple compilers available
    - gcc@11.3.0
    - aocc@4.0.0
    - intel-oneapi-compilers@2024.0.0

    # MPI with each compiler
    - openmpi@4.1.4 %gcc@11.3.0
    - openmpi@4.1.4 %aocc@4.0.0
    - intel-oneapi-mpi@2021.11.0

    # User can choose compiler via modules
    - lammps@20230802 %gcc@11.3.0
    - lammps@20230802 %aocc@4.0.0
    - lammps@20230802 %oneapi@2024.0.0
```

**Module Usage**:
```bash
# Load GCC version
module load gcc/11.3.0
module load openmpi/4.1.4-gcc-11.3.0
module load lammps/20230802-gcc-11.3.0

# Or load AOCC version
module load aocc/4.0.0
module load openmpi/4.1.4-aocc-4.0.0
module load lammps/20230802-aocc-4.0.0

# Or load Intel version
module load intel-oneapi-compilers/2024.0.0
module load intel-oneapi-mpi/2021.11.0
module load lammps/20230802-oneapi-2024.0.0
```

## Spack Compiler Syntax

### AOCC
```yaml
# Install AOCC compiler
- aocc@4.0.0

# Build packages with AOCC
- openmpi@4.1.4 %aocc@4.0.0
- lammps@20230802 %aocc@4.0.0

# Use AOCC libraries
- aocl-blis@4.0      # BLAS
- aocl-libflame@4.0  # LAPACK
- aocl-fftw@4.0      # FFT
```

### Intel oneAPI
```yaml
# Install Intel compilers
- intel-oneapi-compilers@2024.0.0

# Build packages with Intel
- openmpi@4.1.4 %oneapi@2024.0.0
- lammps@20230802 %oneapi@2024.0.0

# Use Intel libraries
- intel-oneapi-mkl@2024.0.0   # Math Kernel Library
- intel-oneapi-tbb@2021.11.0  # Threading Building Blocks
```

## License Considerations

### AOCC
- **License**: Free for use
- **EULA**: Requires acceptance during installation
- **Spack Handling**: Spack can handle EULA acceptance automatically
- **No registration required**

### Intel oneAPI
- **License**: Free for use (community license)
- **Registration**: May require Intel account during first use
- **Spack Handling**: Spack downloads from Intel public repositories
- **No license server required for basic use**

Both compilers are free for HPC use and don't require license servers, making them suitable for cloud deployments.

## Bootstrap Script Changes

The bootstrap script generator needs minimal changes:

```go
// pkg/software/manager.go

// Spack already handles these compilers, just ensure proper package specs
func (m *Manager) GenerateInstallCommands(packages []string) []string {
    var commands []string

    for _, pkg := range packages {
        // Check if it's a compiler package
        if isCompilerPackage(pkg) {
            // Add compiler to Spack after installation
            commands = append(commands, fmt.Sprintf("spack install %s", pkg))
            commands = append(commands, fmt.Sprintf("spack compiler find $(spack location -i %s)", extractPackageName(pkg)))
        } else {
            commands = append(commands, fmt.Sprintf("spack install %s", pkg))
        }
    }

    return commands
}

func isCompilerPackage(pkg string) bool {
    compilers := []string{"gcc", "aocc", "intel-oneapi-compilers", "llvm"}
    pkgName := extractPackageName(pkg)
    for _, c := range compilers {
        if pkgName == c {
            return true
        }
    }
    return false
}
```

## Testing Plan

### 1. AOCC on AMD Instance

```bash
# Create seed with AOCC
cat > test-aocc.yaml <<EOF
cluster:
  name: test-aocc
  region: us-west-2
compute:
  head_node: c6a.large
  queues:
    - name: compute
      instance_types: [c6a.xlarge]
      min_count: 0
      max_count: 2
software:
  spack_packages:
    - aocc@4.0.0
    - openmpi@4.1.4 %aocc@4.0.0
EOF

# Build AMI
petal ami build --seed test-aocc.yaml --name test-aocc-ami

# Create cluster
petal create --seed test-aocc.yaml --custom-ami ami-XXX

# Verify
petal ssh test-aocc
module avail            # Should show aocc, openmpi
module load aocc
clang --version         # Should show AOCC 4.0.0
module load openmpi
mpicc --version         # Should show AOCC-compiled MPI
```

### 2. Intel oneAPI on Intel Instance

```bash
# Create seed with Intel
cat > test-intel.yaml <<EOF
cluster:
  name: test-intel
  region: us-east-1
compute:
  head_node: c6i.large
  queues:
    - name: compute
      instance_types: [c6i.xlarge]
      min_count: 0
      max_count: 2
software:
  spack_packages:
    - intel-oneapi-compilers@2024.0.0
    - intel-oneapi-mpi@2021.11.0
    - intel-oneapi-mkl@2024.0.0
EOF

# Build and test similar to AOCC
```

### 3. Performance Benchmark

```bash
# Build same application with all three compilers
# Compare performance on appropriate instance types

# GCC baseline
module load gcc openmpi
mpicc -O3 benchmark.c -o bench-gcc
sbatch run-bench-gcc.sh

# AOCC on AMD
module load aocc openmpi
mpicc -O3 benchmark.c -o bench-aocc
sbatch run-bench-aocc.sh

# Intel on Intel
module load intel-oneapi-compilers intel-oneapi-mpi
mpicc -O3 benchmark.c -o bench-intel
sbatch run-bench-intel.sh

# Compare results
```

## Documentation Updates

### README.md
Add compiler support section:
```markdown
### Optimized Compilers

petal supports multiple compiler toolchains:

**GCC** (default): Open-source, broad compatibility
- `gcc@11.3.0`, `gcc@12.2.0`

**AOCC**: Optimized for AMD EPYC processors
- `aocc@4.0.0`
- 10-30% faster on AMD instances (c6a, m6a, r6a)
- Free for use

**Intel oneAPI**: Optimized for Intel Xeon processors
- `intel-oneapi-compilers@2024.0.0`
- Includes Math Kernel Library (MKL)
- 10-40% faster on Intel instances (c6i, m6i, r6i)
- Free for use

See `seeds/library/compute-amd-aocc.yaml` and `seeds/library/compute-intel-oneapi.yaml` for examples.
```

### Seed Examples
Create three new library seeds:
- `seeds/library/compute-amd-aocc.yaml`
- `seeds/library/compute-intel-oneapi.yaml`
- `seeds/library/compute-multi-compiler.yaml`

## Benefits

1. **Performance**: 10-30% speedup for HPC workloads
2. **Architecture Optimization**: Match compiler to instance type
3. **No Additional Cost**: Both compilers are free
4. **Industry Standard**: Many HPC sites use Intel/AMD compilers
5. **User Choice**: Let users pick best compiler for their workload

## Acceptance Criteria

- [ ] AOCC compiler installs via Spack
- [ ] Intel oneAPI compilers install via Spack
- [ ] Packages can be built with AOCC (`%aocc@4.0.0`)
- [ ] Packages can be built with Intel (`%oneapi@2024.0.0`)
- [ ] Lmod modules load correctly for each compiler
- [ ] Example seeds created and tested
- [ ] Documentation updated with compiler options
- [ ] Performance benchmarks show expected gains
- [ ] AMI builds complete successfully
- [ ] Multi-compiler environments work

## Related Issues

- Issue #XX: Workload testing (validate compiler performance)
- Future: Auto-select compiler based on instance type
- Future: Compiler comparison benchmarks

## Estimated Effort

- Research Spack compiler support: 1 hour
- Create example seeds: 2 hours
- Update bootstrap script (if needed): 2 hours
- Testing (AOCC + Intel): 4 hours
- Documentation: 2 hours
- Performance benchmarking: 3 hours
- **Total: 14-16 hours**

## References

- [AMD AOCC](https://www.amd.com/en/developer/aocc.html)
- [Intel oneAPI](https://www.intel.com/content/www/us/en/developer/tools/oneapi/overview.html)
- [Spack AOCC Package](https://packages.spack.io/package.html?name=aocc)
- [Spack Intel oneAPI Packages](https://packages.spack.io/package.html?name=intel-oneapi-compilers)
- [AWS AMD Instance Types](https://aws.amazon.com/ec2/instance-types/#Compute_Optimized)
- [AWS HPC Best Practices](https://docs.aws.amazon.com/wellarchitected/latest/high-performance-computing-lens/compilers-and-libraries.html)

## Priority Justification

**High Priority** because:
1. Significant performance improvements (10-30%)
2. Common HPC requirement
3. Low implementation complexity (Spack handles most of it)
4. Free to use (no licensing concerns)
5. Differentiator for petal vs raw ParallelCluster
6. AMD EPYC instances (c6a, hpc7a) are cost-effective for HPC
