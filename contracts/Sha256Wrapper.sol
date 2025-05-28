// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Sha256Wrapper {
    function sha256Hash(bytes memory input) public view returns (bytes32 result) {
        assembly {
            let len := mload(input)
            let ptr := add(input, 0x20)
            let outPtr := mload(0x40)
            if iszero(staticcall(gas(), 0x02, ptr, len, outPtr, 32)) {
                revert(0, 0)
            }
            result := mload(outPtr)
        }
    }
}
