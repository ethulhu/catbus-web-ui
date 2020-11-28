// SPDX-FileCopyrightText: 2020 Benedict Harcourt <ben.harcourt@harcourtprogramming.co.uk>
//
// SPDX-License-Identifier: BSD-2-Clause

'use strict';

const parser  = new DOMParser();
const crypto  = window.crypto.subtle;
const encoder = new TextEncoder();

export function addDefaultHooks()
{
	window.addEventListener( 'click', defaultEventHandler );
	window.addEventListener( 'popstate', e => { loadPage( document.location ) } );
}

export function defaultEventHandler( e )
{
	// Only process on primary buttons.
	if ( e.button !== 0 ) return;

	// Only process when no meta keys are held (allow opening in new tab, etc.)
	if ( e.ctrlKey || e.shiftKey || e.metaKey || e.altKey ) return;

	// Find a link that is being clicked on.
	const link = e.target.closest( 'a' );

	// This function is operating only on anchors.
	if ( ! link ) return;

	// Don't process links with any target=.
	if ( link.hasAttribute( 'target' ) ) return;

	// Don't attempt to process links to nowhere.
	if ( ! link.hasAttribute( 'href' ) ) return;

	// Get (and verify) the link's target.
	const href = link.getAttribute( 'href' );

	// Exclude intra-page anchors.
	if ( href.startsWith( '#' ) ) return;

	// Don't follow external links. Because of how HTML anchors work,
	// all links which are not on the same host name will have to contain
	// a double forward slash token, separating the protocol from the host.
	if ( href.match( /\/\// ) ) return;

	// Check is the client JS wants this page to be turbo loaded.
	// If the event 'turbo:fetch' is cancelled, the link is processed
	// by the default handler.
	const turboFetchEvent = new CustomEvent( 'turbo:fetch', { detail: href } );
	if ( ! link.dispatchEvent( turboFetchEvent ) ) return;

	// We are overriding the default action of clicking on a link.
	e.preventDefault();

	// And we are loading a page.
	loadPage( href );

	// Update the URL bar.
	window.history.pushState( null, null, href );
}

export async function loadPage( href )
{
	// Load the new document and parse the HTML.
	// TODO: This needs error handling. In the event on an
	// error, I think I want to trigger an cancellable event which marks
	// that we're going to send the user to the page. If it is not cancelled,
	// we update `window.location`. (if it is cancelled, nothing happens).
	const newDoc = await fetch( href )
		.then( r => r.text() )
		.then( html => parser.parseFromString( html, 'text/html') );

	// Replace the head and the body of current doc with the new document.
	diffAndImportTree( document.head, newDoc.head );
	diffAndImportTree( document.body, newDoc.body );
}

export async function diffAndImportTree( currentElem, importedElem, digest )
{
	digest = digest || digestSHA;

	const importedHashes = await Promise.all( [...importedElem.children].map( digest ) );

	// The index of the current child in the existing DOM.
	let childIndex = 0;
	// The index of the last item imported from the next list.
	let lastImportedItem = 0;

	// The current node we're operating on.
	let currentNode;
	// The index of the current element in the import list.
	let importedIndex;

	console.log( currentElem.outerHTML, importedElem.outerHTML, importedHashes );

	while ( childIndex < currentElem.childElementCount )
	{
		currentNode   = currentElem.children[ childIndex ];
		importedIndex = importedHashes.indexOf( await digest( currentNode ) );

		console.log(
			'On ', childIndex, currentNode, ' with import ref ', importedIndex,
			'.\n', lastImportedItem, ' of ', importedElem.childElementCount, ' imported'
		);

		// -1 implies this node is not in the import, and so should be deleted.
		// The same is true if this element was already imported.
		if ( importedIndex < lastImportedItem )
		{
			const deleteEvent = new CustomEvent( 'turbo:delete' );
			console.log( 'Requesting to delete', currentNode );

			if ( currentNode.dispatchEvent( deleteEvent ) )
			{
				console.log( 'Deleted', currentNode );
				// We're OK to delete this item.
				currentElem.removeChild( currentNode );

				// We do not need to increment the childIndex,
				// as a different element will be in this position.
				continue;
			}

			console.log( 'Delete was cancelled' );

			// Otherwise, we should not delete. We will leave this element
			// alone, and move on to the next one.
			++childIndex;
			continue;
		}

		// Otherwise, this node should be retained, and any nodes in the
		// import list that are before it should also be imported now.
		// TODO: Do this with less reallocation and also use a document fragment.
		// Note: we import up to `importedIndex - 1` because this node is a
		// clone of the one at `importedIndex`.
		if ( importedIndex > lastImportedItem )
		{
			let n = [...importedElem.children].slice( lastImportedItem, importedIndex );
			console.log( 'Adding elements ', lastImportedItem, importedIndex, n );
				n.forEach( e => currentElem.insertBefore( e.cloneNode( true ), currentNode ) );
		}

		// Update the indices.
		childIndex += importedIndex - lastImportedItem + 1;
		lastImportedItem = importedIndex + 1;

		console.log( 'Keeping element', currentNode, '\ncurrent index', childIndex, ', import index', lastImportedItem );
	}

	console.log( 'Done with the diff phase' );

	// Append any left over nodes.
	if ( lastImportedItem < importedHashes.length - 1 )
	{
		// TODO: Do this with less reallocation and also use a document fragment.
		let n = [...importedElem.children].slice( lastImportedItem );
		console.log( 'Adding remaining elements ', lastImportedItem, n );
			n.forEach( e => currentElem.appendChild( e ) );
	}
}

async function digestSHA( node )
{
	const buffer = await crypto.digest(
		'SHA-1',
		encoder.encode( node.outerHTML )
	);

	return new Uint8Array( buffer ).reduce( (a, v) => a + v.toString( 16 ), '' );
}
