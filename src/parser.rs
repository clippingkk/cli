
#[derive(Debug)]
pub struct TClippingItem {
	title: String,
	content: String,
	pageAt: String,
	createdAt: String,
}

pub enum KindleClippingFileLines {
	Title,
	Info,
	Content,
}

pub fn do_parse() -> Result<Vec<TClippingItem>, String> {
	
	let result = TClippingItem{
		content: "".to_string(),
		title: "".to_string(),
		pageAt: "".to_string(),
		createdAt: "".to_string(),
	};

	let result_list = vec![result];

	Ok(result_list)
}

// const KindleDateTimeLayout {
// 	ENLayout = "Monday, January 2, 2006 3:4:5 PM"
// 	KindleDateTimeZHLayout = "2006-1-2 3:4:5"
// }
