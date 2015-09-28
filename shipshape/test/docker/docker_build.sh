#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Script that build a test Docker image for Shipshape and then starts a
# container using the image.

set -eu

if [ $# -lt 2 ]; then
  echo "Usage: $0 <CONTAINER> <IMAGE>"
  exit 1
fi

declare -xr TEST_DIR=$(realpath $(dirname "$0"))
declare -xr CONTAINER="$1"
declare -xr IMAGE="$2"

echo " Building docker image ... "
docker build -f "${TEST_DIR}/Dockerfile" -t "${IMAGE}" "${TEST_DIR}" || exit 1

echo " Remove running container [${CONTAINER}] "
CONTAINER_ID=`docker ps -a | grep "${CONTAINER}" | awk '{print $1}'`

echo " Found container id: ${CONTAINER_ID} "
if [ -n "${CONTAINER_ID}" ]
then
  docker rm -f "${CONTAINER_ID}"
fi

exit 0
