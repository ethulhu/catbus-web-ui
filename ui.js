'use strict';

import { elemRegister } from './elems.js';
elemRegister( '_', null, 'button', 'div', 'input', 'h2', 'h3', 'label', 'main', 'option', 'p', 'section', 'select', 'span', 'table', 'td', 'tr' );

Array.prototype.last = function() {
	return this[ this.length - 1 ];
}
Array.prototype.unique = function() {
	return this.filter( ( x, i ) => this.indexOf( x ) === i );
}
Element.prototype.insertSortedById = function( newChild ) {
        for ( const child of this.children ) {
		if ( ! child.id ) {
			continue;
		}
                if ( newChild.id < child.id ) {
                        this.insertBefore( newChild, child );
                        return;
                }
        }
        this.appendChild( newChild );
}
Map.prototype.sortedKeys = function() {
	const keys = Array.from( this.keys() );
	keys.sort();
	return keys;
}
Map.prototype.getOrDefault = function( key, defaultValue ) {
	return this.has( key ) ? this.get( key ) : defaultValue;
}
String.prototype.toTitleCase = function() {
	return this.split( ' ' )
		   .map( word => `${word.charAt( 0 ).toUpperCase()}${word.slice( 1 )}` )
		   .join( ' ' );
}

const nameForId = id => id.split( '/' ).last()
			  .split( '_' )[0] 
			  .replace( '-', ' ' );

const isControl = topic => topic.split( '/' ).length === 4;

export class UI {
	constructor( { rootElement, topicPrefix, sendMessageFunction } ) {
		if ( ! rootElement )         { throw 'rootElement must be set'; }
		if ( ! topicPrefix )         { throw 'topicPrefix must be set'; }
		if ( ! sendMessageFunction ) { throw 'sendMessageFunction must be set'; }

		this.rootElement = rootElement;
		this.topicPrefix = topicPrefix;
		this.sendMessage = sendMessageFunction;

		this.values   = new Map();
		this.updaters = new Map();
	}

	set( topic, value ) {
		const newControl = ( ! this.values.has( topic ) ) && isControl( topic );

		this.values.set( topic, value );

		if ( newControl ) {
			const [ home, zone, device, control ] = topic.split( '/' );
			// TODO: maybe bail if home !== this.topicPrefix ?
			let zoneSection = document.getElementById( `${home}/${zone}` );
			if ( ! zoneSection ) {
				zoneSection = _section(
					{ id: `${home}/${zone}` },
					_h2( nameForId( zone ) ),
					_button(
						'everything off',
						{ click: e => Array.from( this.values.keys() )
						                   .filter( k => new RegExp( `^${home}/${zone}/[^/]+/power$` ).test( k ) )
						                   .map( k => this.sendMessage( k, 'off' ) ) },
					),
				);
				this.rootElement.insertSortedById( zoneSection );
			}
			let deviceSection = document.getElementById( `${home}/${zone}/${device}` );
			if ( ! deviceSection ) {
				deviceSection = _section(
						{ id: `${home}/${zone}/${device}` },
						_h3( nameForId( device ) ),
						_table(),
				);
				zoneSection.insertSortedById( deviceSection );
			}

			const controlElement = this.controlForTopic( topic );
			deviceSection.querySelector( 'table' ).insertSortedById( controlElement );
		}

		if ( this.updaters.has( topic ) ) {
			this.updaters.get( topic )( topic, value );
		} else {
			const updater = this.updaterForTopic( topic );
			if ( updater ) {
				this.updaters.set( topic, updater );
				updater( topic, value );
			}
		}
	}

	updaterForTopic( topic ) {
	}

	get rooms() {
		return this.values.sortedKeys()
		                  .filter( k => k.startsWith( `${this.topicPrefix}/` ) )
		                  .map( k => k.split( '/' )[ 1 ] )
		                  .unique();
	}
	devices( room ) {
		return this.values.sortedKeys()
		                  .filter( k => k.startsWith( `${room}/` ) )
		                  .map( k => k.split( '/' )[ 2 ] )
		                  .unique();
	}
	controls( device ) {
		return this.values.sortedKeys()
		                  .filter( k => k.startsWith( `${device}/` ) )
		                  .map( k => k.split( '/' )[ 3 ] )
		                  .unique();
	}

	updaterForTopic( topic ) {
		const _enum = ( topic, value ) => {
			const enumElement = document.getElementById( topic ).querySelector( 'select' );

			// the enum's value needs to be in the options to be visible,
			// so add it if it's not there already.
			let ok = false;
			for ( const option of enumElement.options ) {
				ok = ok || ( option.text === value );
			}
			if ( ! ok ) {
				enumElement.appendChild( _option( value ) );
			}

			enumElement.value = value;
		};
		const _enumValues = ( topic, value ) => {
			const enumTopic  = topic.slice( 0, - '/values'.length )
			if ( ! this.values.has( enumTopic ) ) {
				return
			}
			const enumValue   = this.values.get( enumTopic );

			const enumParent  = document.getElementById( enumTopic )
			const enumElement = enumParent.querySelector( 'select' );

			// the enum's value needs to be in the options to be visible.
			// so add it if it's not there already.
			const values = value.split( '\n' );
			if ( ! values.includes( enumValue ) ) {
				values.push( enumValue );
			}

			enumElement.innerHTML = '';
			values.forEach( value => {
				enumElement.appendChild( _option( value ) );
			} );
			enumElement.value = enumValue;
		};
		const _power      = ( topic, value ) => { document.getElementById( topic ).querySelector( 'input' ).checked = value === 'on'; };
		const _range      = ( topic, value ) => { document.getElementById( topic ).querySelector( 'input' ).value = value; };
		const _sensor     = ( topic, value ) => { document.getElementById( topic ).querySelector( 'span'  ).textContent = value; };
		const _text       = ( topic, value ) => { document.getElementById( topic ).querySelector( 'input' ).value = value; };

		return topic.includes( 'hygrothermograph' ) ? _sensor :
		       topic.endsWith( 'degrees' )     ? _range      :
		       topic.endsWith( 'enum' )        ? _enum       :
		       topic.endsWith( 'enum/values' ) ? _enumValues :
		       topic.endsWith( 'kelvin' )      ? _range      :
		       topic.endsWith( 'percent' )     ? _range      :
		       topic.endsWith( 'power' )       ? _power      :
		                                         _text       ;
	}

	controlForTopic( topic ) {
		const _power = id => _tr(
			{ id: id },
			_td( 'power' ),
			_td( _input( {
				type: 'checkbox',
				checked: this.values.get( id ) === 'on',
				change: e => this.sendMessage(
					id,
					e.target.checked ? 'on' : 'off',
				),
			} ) ),
		);

		const _range = ( id, min, max ) => _tr(
			{ id: id },
			_td( nameForId( id ) ),
			_td( _input( {
				type: 'range',
				min: min,
				max: max,
				value: this.values.get( id ),
				change: e => this.sendMessage( id, e.target.value ),
			} ) ),
		);
		const _percent = id => _range( id, 0,    100  );
		const _degrees = id => _range( id, 0,    359  );
		const _kelvin  = id => _range( id, 2500, 9000 );

		const _text = id => _tr(
			{ id: id },
			_td( nameForId( id ) ),
			_td( _input( {
				type: 'text',
				value: this.values.get( id ),
				change: e => this.sendMessage( id, e.target.value ),
			} ) ),
		);

		const _enum = id => _tr(
			{ id: id },
			_td( nameForId( id ) ),
			_td( _select(
				{ change: e => this.sendMessage( id, e.target.value ) },
				this.values.getOrDefault( `${id}/values`, this.values.get( id ) )  // always include our current value.
				           .split( '\n' ).map( value => _option(
						value,
						{ selected: value === this.values.get( id ) },
				) ),
			) ),
		);

		const _sensor = id => {
			const unit =
				id.endsWith( 'celsius' ) ? '°C' :
				id.endsWith( 'percent' ) ? '%'  :
							   ''   ;
			return _tr(
				{ id: id },
				_td( nameForId( id ) ),
				_td(
					_span( this.values.get( id ) ),
					unit,
				),
			);
		};

		return topic.includes( 'hygrothermograph' ) ? _sensor( topic ) :
			topic.endsWith( 'degrees' ) ? _degrees( topic ) :
			topic.endsWith( 'enum' )    ? _enum( topic )    :
			topic.endsWith( 'kelvin' )  ? _kelvin( topic )  :
			topic.endsWith( 'percent' ) ? _percent( topic ) :
			topic.endsWith( 'power' )   ? _power( topic )   :
						   _text( topic )    ;
	}
}
