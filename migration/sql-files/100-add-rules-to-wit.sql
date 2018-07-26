-- Add action rules fields to work_item_types. The values are
-- optional, so they may be null and are being kept at null
-- on existing WITs. Values for current templates will be
-- imported from the template on next launch.
ALTER TABLE work_item_types ADD COLUMN trans_rule_key text;
ALTER TABLE work_item_types ADD COLUMN trans_rule_argument text;

-- migrate existing work items

-- Agile Template - Impediment
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["7389fa7d-39c8-4865-8094-eda9a7836161", "7e3bbf09-44c4-419e-8d43-10e00400ca80"] }' WHERE fields @> '{"system.state": "New"}' AND type="03b9bb64-4f65-4fa7-b165-494cd4f01401";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["7063ae46-994d-49e8-99f9-2ad867dd340e", "29124ef0-d651-47c4-84a7-28acb7a4ab7a"] }' WHERE fields @> '{"system.state": "Open"}' AND type="03b9bb64-4f65-4fa7-b165-494cd4f01401";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["f7243e68-1d2b-4256-b6e7-3c657c944ff1", "a30fc0e0-bfa9-43b1-a83d-b62ae2d5d0f7"] }' WHERE fields @> '{"system.state": "In Progress"}' AND type="03b9bb64-4f65-4fa7-b165-494cd4f01401";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["9f780106-4d71-41bf-b017-001ca7e19162", "ca1ea842-1650-4435-88b3-560e5bf47d42"] }' WHERE fields @> '{"system.state": "Resolved"}' AND type="03b9bb64-4f65-4fa7-b165-494cd4f01401";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "Closed"}' AND type="03b9bb64-4f65-4fa7-b165-494cd4f01401";
-- Agile Template - Defect
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["7389fa7d-39c8-4865-8094-eda9a7836161, 7e3bbf09-44c4-419e-8d43-10e00400ca80"] }' WHERE fields @> '{"system.state": "New"}' AND type="fce0921f-ea70-4513-bb91-31d3aa8017f1";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["7063ae46-994d-49e8-99f9-2ad867dd340e, 29124ef0-d651-47c4-84a7-28acb7a4ab7a"] }' WHERE fields @> '{"system.state": "Open"}' AND type="fce0921f-ea70-4513-bb91-31d3aa8017f1";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["f7243e68-1d2b-4256-b6e7-3c657c944ff1, a30fc0e0-bfa9-43b1-a83d-b62ae2d5d0f7"] }' WHERE fields @> '{"system.state": "In Progress"}' AND type="fce0921f-ea70-4513-bb91-31d3aa8017f1";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["9f780106-4d71-41bf-b017-001ca7e19162, ca1ea842-1650-4435-88b3-560e5bf47d42"] }' WHERE fields @> '{"system.state": "Resolved"}' AND type="fce0921f-ea70-4513-bb91-31d3aa8017f1";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "Closed"}' AND type="fce0921f-ea70-4513-bb91-31d3aa8017f1";
-- Agile Template - Task
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["7389fa7d-39c8-4865-8094-eda9a7836161, 7e3bbf09-44c4-419e-8d43-10e00400ca80"] }' WHERE fields @> '{"system.state": "New"}' AND type="2853459d-60ef-4fbe-aaf4-eccb9f554b34";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["7063ae46-994d-49e8-99f9-2ad867dd340e, 29124ef0-d651-47c4-84a7-28acb7a4ab7a"] }' WHERE fields @> '{"system.state": "Open"}' AND type="2853459d-60ef-4fbe-aaf4-eccb9f554b34";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["f7243e68-1d2b-4256-b6e7-3c657c944ff1, a30fc0e0-bfa9-43b1-a83d-b62ae2d5d0f7"] }' WHERE fields @> '{"system.state": "In Progress"}' AND type="2853459d-60ef-4fbe-aaf4-eccb9f554b34";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["9f780106-4d71-41bf-b017-001ca7e19162, ca1ea842-1650-4435-88b3-560e5bf47d42"] }' WHERE fields @> '{"system.state": "Resolved"}' AND type="2853459d-60ef-4fbe-aaf4-eccb9f554b34";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "Closed"}' AND type="2853459d-60ef-4fbe-aaf4-eccb9f554b34";
-- Agile Template - Story
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["7389fa7d-39c8-4865-8094-eda9a7836161, 7e3bbf09-44c4-419e-8d43-10e00400ca80"] }' WHERE fields @> '{"system.state": "New"}' AND type="6ff83406-caa7-47a9-9200-4ca796be11bb";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["7063ae46-994d-49e8-99f9-2ad867dd340e, 29124ef0-d651-47c4-84a7-28acb7a4ab7a"] }' WHERE fields @> '{"system.state": "Open"}' AND type="6ff83406-caa7-47a9-9200-4ca796be11bb";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["f7243e68-1d2b-4256-b6e7-3c657c944ff1, a30fc0e0-bfa9-43b1-a83d-b62ae2d5d0f7"] }' WHERE fields @> '{"system.state": "In Progress"}' AND type="6ff83406-caa7-47a9-9200-4ca796be11bb";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["9f780106-4d71-41bf-b017-001ca7e19162, ca1ea842-1650-4435-88b3-560e5bf47d42"] }' WHERE fields @> '{"system.state": "Resolved"}' AND type="6ff83406-caa7-47a9-9200-4ca796be11bb";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "Closed"}' AND type="6ff83406-caa7-47a9-9200-4ca796be11bb";
-- Agile Template - Epic
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["7389fa7d-39c8-4865-8094-eda9a7836161, 7e3bbf09-44c4-419e-8d43-10e00400ca80"] }' WHERE fields @> '{"system.state": "New"}' AND type="2c169431-a55d-49eb-af74-cc19e895356f";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["7063ae46-994d-49e8-99f9-2ad867dd340e, 29124ef0-d651-47c4-84a7-28acb7a4ab7a"] }' WHERE fields @> '{"system.state": "Open"}' AND type="2c169431-a55d-49eb-af74-cc19e895356f";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["f7243e68-1d2b-4256-b6e7-3c657c944ff1, a30fc0e0-bfa9-43b1-a83d-b62ae2d5d0f7"] }' WHERE fields @> '{"system.state": "In Progress"}' AND type="2c169431-a55d-49eb-af74-cc19e895356f";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["9f780106-4d71-41bf-b017-001ca7e19162, ca1ea842-1650-4435-88b3-560e5bf47d42"] }' WHERE fields @> '{"system.state": "Resolved"}' AND type="2c169431-a55d-49eb-af74-cc19e895356f";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "Closed"}' AND type="2c169431-a55d-49eb-af74-cc19e895356f";
-- Agile Template - Theme
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["7389fa7d-39c8-4865-8094-eda9a7836161, 7e3bbf09-44c4-419e-8d43-10e00400ca80"] }' WHERE fields @> '{"system.state": "New"}' AND type="5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["7063ae46-994d-49e8-99f9-2ad867dd340e, 29124ef0-d651-47c4-84a7-28acb7a4ab7a"] }' WHERE fields @> '{"system.state": "Open"}' AND type="5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["f7243e68-1d2b-4256-b6e7-3c657c944ff1, a30fc0e0-bfa9-43b1-a83d-b62ae2d5d0f7"] }' WHERE fields @> '{"system.state": "In Progress"}' AND type="5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["9f780106-4d71-41bf-b017-001ca7e19162, ca1ea842-1650-4435-88b3-560e5bf47d42"] }' WHERE fields @> '{"system.state": "Resolved"}' AND type="5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "Closed"}' AND type="5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a";

-- Legacy(SDD) Template - Task
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["8faebb8a-3748-44c6-a691-27633dde571c"] }' WHERE fields @> '{"system.state": "new"}' AND type="bbf35418-04b6-426c-a60b-7f80beb0b624";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["907dad6c-f117-4ad6-b6dd-e21fb198e56d"] }' WHERE fields @> '{"system.state": "open"}' AND type="bbf35418-04b6-426c-a60b-7f80beb0b624";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["90a0a0b1-3e9c-4921-8430-25ff56fd1996"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="bbf35418-04b6-426c-a60b-7f80beb0b624";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["86a2aaaa-4a80-433b-b390-b8f42eec2d32"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="bbf35418-04b6-426c-a60b-7f80beb0b624";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="bbf35418-04b6-426c-a60b-7f80beb0b624";
-- Legacy(SDD) Template - Bug
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["6c314706-f562-494d-91b9-b7d2c36672ba, 8faebb8a-3748-44c6-a691-27633dde571c"] }' WHERE fields @> '{"system.state": "new"}' AND type="26787039-b68f-4e28-8814-c2f93be1ef4e";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["6b06a763-cdef-400e-98d3-8db46e633c92, 907dad6c-f117-4ad6-b6dd-e21fb198e56d"] }' WHERE fields @> '{"system.state": "open"}' AND type="26787039-b68f-4e28-8814-c2f93be1ef4e";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["92f48297-062b-4605-9f30-2e546af4d898, 90a0a0b1-3e9c-4921-8430-25ff56fd1996"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="26787039-b68f-4e28-8814-c2f93be1ef4e";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["572c67ef-c550-4084-bd8a-a6d722a3278a, 86a2aaaa-4a80-433b-b390-b8f42eec2d32"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="26787039-b68f-4e28-8814-c2f93be1ef4e";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="26787039-b68f-4e28-8814-c2f93be1ef4e";
-- Legacy(SDD) Template - Feature
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["6c314706-f562-494d-91b9-b7d2c36672ba, 8faebb8a-3748-44c6-a691-27633dde571c"] }' WHERE fields @> '{"system.state": "new"}' AND type="0a24d3c2-e0a6-4686-8051-ec0ea1915a28";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["6b06a763-cdef-400e-98d3-8db46e633c92, 907dad6c-f117-4ad6-b6dd-e21fb198e56d"] }' WHERE fields @> '{"system.state": "open"}' AND type="0a24d3c2-e0a6-4686-8051-ec0ea1915a28";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["92f48297-062b-4605-9f30-2e546af4d898, 90a0a0b1-3e9c-4921-8430-25ff56fd1996"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="0a24d3c2-e0a6-4686-8051-ec0ea1915a28";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["572c67ef-c550-4084-bd8a-a6d722a3278a, 86a2aaaa-4a80-433b-b390-b8f42eec2d32"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="0a24d3c2-e0a6-4686-8051-ec0ea1915a28";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="0a24d3c2-e0a6-4686-8051-ec0ea1915a28";
-- Legacy(SDD) Template - Experience
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["8faebb8a-3748-44c6-a691-27633dde571c"] }' WHERE fields @> '{"system.state": "new"}' AND type="b9a71831-c803-4f66-8774-4193fffd1311";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["907dad6c-f117-4ad6-b6dd-e21fb198e56d"] }' WHERE fields @> '{"system.state": "open"}' AND type="b9a71831-c803-4f66-8774-4193fffd1311";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["90a0a0b1-3e9c-4921-8430-25ff56fd1996"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="b9a71831-c803-4f66-8774-4193fffd1311";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["86a2aaaa-4a80-433b-b390-b8f42eec2d32"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="b9a71831-c803-4f66-8774-4193fffd1311";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="b9a71831-c803-4f66-8774-4193fffd1311";
-- Legacy(SDD) Template - ValueProposition
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["8faebb8a-3748-44c6-a691-27633dde571c"] }' WHERE fields @> '{"system.state": "new"}' AND type="3194ab60-855b-4155-9005-9dce4a05f1eb";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["907dad6c-f117-4ad6-b6dd-e21fb198e56d"] }' WHERE fields @> '{"system.state": "open"}' AND type="3194ab60-855b-4155-9005-9dce4a05f1eb";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["90a0a0b1-3e9c-4921-8430-25ff56fd1996"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="3194ab60-855b-4155-9005-9dce4a05f1eb";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["86a2aaaa-4a80-433b-b390-b8f42eec2d32"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="3194ab60-855b-4155-9005-9dce4a05f1eb";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="3194ab60-855b-4155-9005-9dce4a05f1eb";
-- Legacy(SDD) Template - Scenario
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["b4edad70-1d77-4e5a-b973-0f0d599fd20d"] }' WHERE fields @> '{"system.state": "new"}' AND type="71171e90-6d35-498f-a6a7-2083b5267c18";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["ce5cd7bd-1eb3-4945-821f-ebfedebf5958"] }' WHERE fields @> '{"system.state": "open"}' AND type="71171e90-6d35-498f-a6a7-2083b5267c18";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["42120527-5a99-4913-9917-58450008b770"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="71171e90-6d35-498f-a6a7-2083b5267c18";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["b7ef0df4-2253-47ee-9e60-4f768a5d7c81"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="71171e90-6d35-498f-a6a7-2083b5267c18";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="71171e90-6d35-498f-a6a7-2083b5267c18";
-- Legacy(SDD) Template - Fundamental
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["b4edad70-1d77-4e5a-b973-0f0d599fd20d"] }' WHERE fields @> '{"system.state": "new"}' AND type="ee7ca005-f81d-4eea-9b9b-1965df0988d0";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["ce5cd7bd-1eb3-4945-821f-ebfedebf5958"] }' WHERE fields @> '{"system.state": "open"}' AND type="ee7ca005-f81d-4eea-9b9b-1965df0988d0";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["42120527-5a99-4913-9917-58450008b770"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="ee7ca005-f81d-4eea-9b9b-1965df0988d0";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["b7ef0df4-2253-47ee-9e60-4f768a5d7c81"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="ee7ca005-f81d-4eea-9b9b-1965df0988d0";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="ee7ca005-f81d-4eea-9b9b-1965df0988d0";
-- Legacy(SDD) Template - Papercut
update work_items set fields = fields || '{"system.metastate": "mNew", "system.boardcolumns": ["b4edad70-1d77-4e5a-b973-0f0d599fd20d"] }' WHERE fields @> '{"system.state": "new"}' AND type="6d603ab4-7c5e-4c5f-bba8-a3ba9d370985";
update work_items set fields = fields || '{"system.metastate": "mOpen", "system.boardcolumns": ["ce5cd7bd-1eb3-4945-821f-ebfedebf5958"] }' WHERE fields @> '{"system.state": "open"}' AND type="6d603ab4-7c5e-4c5f-bba8-a3ba9d370985";
update work_items set fields = fields || '{"system.metastate": "mInprogress", "system.boardcolumns": ["42120527-5a99-4913-9917-58450008b770"] }' WHERE fields @> '{"system.state": "in Progress"}' AND type="6d603ab4-7c5e-4c5f-bba8-a3ba9d370985";
update work_items set fields = fields || '{"system.metastate": "mResolved", "system.boardcolumns": ["b7ef0df4-2253-47ee-9e60-4f768a5d7c81"] }' WHERE fields @> '{"system.state": "resolved"}' AND type="6d603ab4-7c5e-4c5f-bba8-a3ba9d370985";
update work_items set fields = fields || '{"system.metastate": "mClosed", "system.boardcolumns": [] }' WHERE fields @> '{"system.state": "closed"}' AND type="6d603ab4-7c5e-4c5f-bba8-a3ba9d370985";
