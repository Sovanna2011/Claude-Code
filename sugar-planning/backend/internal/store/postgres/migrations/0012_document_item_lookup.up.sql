-- The document list, filtered to one warehouse, asks "which documents touched
-- this store?" through an EXISTS on the item table. The existing index is
-- (warehouse_id, product_id), which answers "what is in this store" but makes
-- the semi-join fetch every matching row from the heap to read its document_id.
--
-- Adding document_id lets that be an index-only scan. Measured on 616,511
-- items: the heap fetches go from 153,450 to zero.
--
-- This is not the whole fix. The count beside the page was the larger cost and
-- is bounded in the query rather than indexed away - see store.CountLimit.
CREATE INDEX inventory_document_items_warehouse_doc_idx
    ON inventory_document_items (warehouse_id, document_id);
