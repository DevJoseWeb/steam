/*
   Steam Library For Go
   Copyright (C) 2016 Ahmed Samy <f.fallen45@gmail.com>

   This library is free software; you can redistribute it and/or
   modify it under the terms of the GNU Lesser General Public
   License as published by the Free Software Foundation; either
   version 2.1 of the License, or (at your option) any later version.

   This library is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
   Lesser General Public License for more details.

   You should have received a copy of the GNU Lesser General Public
   License along with this library; if not, write to the Free Software
   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// Due to the JSON being string, etc... we cannot re-use EconItem
// Also, "assetid" is included as "id" not as assetid.
type InventoryItem struct {
	AssetID        uint64 `json:"id,string,omitempty"`
	InstanceID     uint64 `json:"instanceid,string,omitempty"`
	ClassID        uint64 `json:"classid,string,omitempty"`
	AppID          uint32 `json:"appid"`     // This!
	ContextID      uint16 `json:"contextid"` // Ditto
	Name           string `json:"name"`
	MarketHashName string `json:"market_hash_name"`
	Pos            uint32 `json:"pos"` // Needed to match with item description in inventory, see below.
}

var (
	ErrCannotLoadInventory = errors.New("unable to load inventory at this time")
)

func (community *Community) GetInventory(sid *SteamID, appid, contextid uint32, tradableOnly bool) ([]*InventoryItem, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://steamcommunity.com/profiles/%d/inventory/json/%d/%d/?trading=%d",
			*sid, appid, contextid, tradableOnly),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := community.client.Do(req)
	if err != nil {
		return nil, err
	}

	type DescItem struct {
		Name           string `json:"name"`
		MarketName     string `json:"market_name"` // Purge?
		MarketHashName string `json:"market_hash_name"`
	}

	type Response struct {
		Success      bool                      `json:"success"`
		Inventory    map[string]*InventoryItem `json:"rgInventory"`
		Descriptions map[string]*DescItem      `json:"rgDescriptions"`
		/* Missing: rgCurrency  */
	}

	var r Response
	if err = json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	if !r.Success {
		return nil, ErrCannotLoadInventory
	}

	// Morph r.Inventory into an array of items.
	// This is due to Steam returning the items in the following format:
	//	rgInventory: {
	//		"54xxx": {
	//			"id": "54xxx"
	//			...
	//		}
	//	}
	// FIXME: descriptions broken
	items := []*InventoryItem{}
	for _, value := range r.Inventory {
		if value.Pos < uint32(len(r.Descriptions)) {
			desc, ok := r.Descriptions[strconv.FormatUint(value.AssetID, 10)+"_"+strconv.FormatUint(value.InstanceID, 10)]
			if ok {
				value.Name = desc.Name
				value.MarketHashName = desc.MarketHashName
			}
		}

		items = append(items, value)
	}

	return items, nil
}
