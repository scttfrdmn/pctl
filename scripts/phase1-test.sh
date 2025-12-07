#!/bin/bash
# Phase 1 Workload Testing Script
# Tests basic cluster functionality and validates software installation

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

CLUSTER_NAME="workload-test-basic"
KEY_NAME="petal-testing-key"
SEED_FILE="seeds/testing/workload-basic.yaml"

echo -e "${BLUE}🧪 Phase 1: Basic Cluster Functionality Test${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Step 1: Create cluster
echo -e "${YELLOW}Step 1: Creating cluster...${NC}"
echo "  Cluster: $CLUSTER_NAME"
echo "  Region: us-west-2"
echo "  Instance: t3.medium head, t3.small compute"
echo ""

petal create --seed "$SEED_FILE" \
  --name "$CLUSTER_NAME" \
  --key-name "$KEY_NAME"

echo ""
echo -e "${GREEN}✅ Cluster created!${NC}"
echo ""

# Step 2: Wait for cluster to be ready
echo -e "${YELLOW}Step 2: Waiting for cluster to be ready...${NC}"
echo "  This may take 5-10 minutes for initial bootstrap"
echo ""

sleep 60  # Give it a minute to start

# Check status
petal status "$CLUSTER_NAME"

echo ""
echo -e "${BLUE}📝 Manual validation required${NC}"
echo -e "${BLUE}============================${NC}"
echo ""
echo "Once the cluster is CREATE_COMPLETE, please SSH in and run the following tests:"
echo ""
echo -e "${YELLOW}1. SSH to cluster:${NC}"
echo "   petal ssh $CLUSTER_NAME"
echo ""
echo -e "${YELLOW}2. Verify software installation:${NC}"
cat <<'EOF'
   # Check modules are available
   module avail

   # Should show:
   # - gcc/11.3.0
   # - openmpi/4.1.4
   # - python/3.10

   # Load and test GCC
   module load gcc/11.3.0
   gcc --version

   # Load and test OpenMPI
   module load openmpi/4.1.4
   mpirun --version

   # Load and test Python
   module load python/3.10
   python3 --version
EOF

echo ""
echo -e "${YELLOW}3. Verify SLURM:${NC}"
cat <<'EOF'
   sinfo    # Should show compute queue
   squeue   # Should be empty
EOF

echo ""
echo -e "${YELLOW}4. Run SLURM job on compute node (CRITICAL TEST):${NC}"
cat <<'EOF'
   # Create test script
   cat > test.sh <<SCRIPT
#!/bin/bash
#SBATCH -J test
#SBATCH -p compute
#SBATCH -n 1
#SBATCH -t 00:02:00

echo "=== Hello from SLURM ==="
hostname
echo ""

echo "=== Testing /opt/spack access ==="
ls -la /opt/spack || echo "ERROR: /opt/spack not found!"
echo ""

echo "=== Testing module system ==="
module avail
echo ""

echo "=== Testing GCC ==="
module load gcc/11.3.0
gcc --version || echo "ERROR: gcc not found!"
echo ""

echo "=== Testing OpenMPI ==="
module load openmpi/4.1.4
mpirun --version || echo "ERROR: mpirun not found!"
SCRIPT

   # Submit job
   sbatch test.sh

   # Monitor job
   squeue   # Should show job

   # Wait for completion (2 minutes max)
   sleep 130

   # Check output
   cat slurm-*.out

   # If you see "ERROR" in output, compute nodes CANNOT access /opt/spack
   # This means we MUST implement EBS snapshot architecture immediately

   # If output shows version numbers, our current approach works!
EOF

echo ""
echo -e "${YELLOW}5. Verify user creation:${NC}"
cat <<'EOF'
   id testuser
   # Should show: uid=5001(testuser) gid=5001(testuser)
EOF

echo ""
echo -e "${YELLOW}6. Cleanup (run after testing):${NC}"
echo "   exit  # Exit SSH session"
echo "   petal delete $CLUSTER_NAME"
echo ""

echo -e "${BLUE}🎯 Success Criteria:${NC}"
echo "  - [ ] All modules load without errors"
echo "  - [ ] Software versions match specifications"
echo "  - [ ] SLURM job completes successfully"
echo "  - [ ] Compute node can access /opt/spack"
echo "  - [ ] User exists with correct UID/GID"
echo "  - [ ] Total cost < $3"
echo ""

echo -e "${RED}⚠️  CRITICAL:${NC} The SLURM job test (step 4) answers the fundamental question:"
echo "  ${YELLOW}Can compute nodes access software installed at /opt/spack?${NC}"
echo ""
echo "  - If YES: Current AMI approach works, proceed with Phase 2"
echo "  - If NO:  Must implement EBS snapshot architecture immediately"
echo ""
